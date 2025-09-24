package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/store/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// scheduler 完全事件驱动的分布式任务调度器
type scheduler struct {
	// 基本信息
	nodeID  string
	options *SchedulerOptions

	// ETCD组件
	etcdClient *etcd.Etcd
	session    *concurrency.Session
	election   *concurrency.Election

	// 状态管理
	mu        sync.RWMutex
	isStarted bool
	isLeader  bool

	// 负载均衡器
	loadBalancer LoadBalancer

	// 事件系统
	eventBus       *EventBus
	eventCallbacks map[EventType][]EventCallback

	// 事件处理器
	eventProcessors map[string]*EventProcessor

	// 租约管理
	leaderLease clientv3.LeaseID

	// 控制通道
	stopChan chan struct{}
	wg       sync.WaitGroup

	// 指标
	metrics   *SchedulerMetrics
	startTime int64
}

// NewScheduler 创建完全事件驱动的分布式调度器实例
func NewScheduler(etcdClient *etcd.Etcd, options *SchedulerOptions) (Scheduler, error) {
	if options == nil {
		options = DefaultSchedulerOptions()
	}

	// 生成节点ID
	if options.NodeID == "" {
		options.NodeID = generateNodeID()
	}

	// 创建具有动态扩容功能的事件总线
	eventBus := NewEventBus(5000)

	// 创建调度器实例
	scheduler := &scheduler{
		options:         options,
		nodeID:          options.NodeID,
		etcdClient:      etcdClient,
		eventBus:        eventBus,
		eventCallbacks:  make(map[EventType][]EventCallback),
		eventProcessors: make(map[string]*EventProcessor),
		stopChan:        make(chan struct{}),
		metrics:         &SchedulerMetrics{},
	}

	// 初始化负载均衡器
	scheduler.loadBalancer = NewLoadBalancer(options.LoadBalanceStrategy, scheduler, nil)

	// 创建事件处理器
	scheduler.eventProcessors["task"] = NewEventProcessor("task", scheduler)
	scheduler.eventProcessors["worker"] = NewEventProcessor("worker", scheduler)
	scheduler.eventProcessors["scheduler"] = NewEventProcessor("scheduler", scheduler)

	return scheduler, nil
}

// Start 启动事件驱动调度器
func (s *scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isStarted {
		return ErrSchedulerAlreadyStarted
	}

	s.startTime = currentTimestamp()

	log.Info().
		Str("nodeId", s.nodeID).
		Msg("starting scheduler")

	// 创建ETCD会话
	session, err := concurrency.NewSession(s.etcdClient.Client, concurrency.WithTTL(int(s.options.ElectionTimeout.Seconds())))
	if err != nil {
		return fmt.Errorf("failed to create etcd session: %w", err)
	}
	s.session = session

	// 启动Leader选举
	if err := s.startLeaderElection(ctx); err != nil {
		return fmt.Errorf("failed to start leader election: %w", err)
	}

	// 启动事件处理器
	for name, processor := range s.eventProcessors {
		if err := processor.Start(); err != nil {
			return fmt.Errorf("failed to start event processor %s: %w", name, err)
		}
	}

	// 启动ETCD监听器
	s.wg.Add(1)
	go s.runETCDWatcher(ctx)

	// 注意：移除定期调度循环，完全依赖事件驱动调度

	// 启动健康检查
	s.wg.Add(1)
	go s.runHealthCheck(ctx)

	// 启动待处理任务检查（低频率检查，防止事件遗漏导致任务积压）
	s.wg.Add(1)
	go s.runPendingTask(ctx)

	// 启动指标收集
	if s.options.EnableMetrics {
		s.wg.Add(1)
		go s.runMetricsCollection(ctx)
	}

	s.isStarted = true

	// 发布调度器启动事件
	s.publishEvent(&SchedulerEvent{
		Type:      EventSchedulerStarted,
		Timestamp: currentTimestamp(),
		Data: map[string]any{
			"nodeId": s.nodeID,
		},
	})

	log.Info().
		Str("nodeId", s.nodeID).
		Msg("scheduler started successfully")

	return nil
}

// Stop 停止事件驱动调度器
func (s *scheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isStarted {
		return ErrSchedulerNotStarted
	}

	log.Info().Str("nodeId", s.nodeID).Msg("stopping scheduler")

	// 停止接收新任务
	close(s.stopChan)

	// 发布调度器停止事件
	s.publishEvent(&SchedulerEvent{
		Type:      EventSchedulerStopped,
		Timestamp: currentTimestamp(),
		Data: map[string]any{
			"nodeId": s.nodeID,
		},
	})

	// 停止事件处理器
	for _, processor := range s.eventProcessors {
		if err := processor.Stop(); err != nil {
			log.Error().Err(err).Msg("failed to stop event processor")
		}
	}

	// 停止事件总线
	if err := s.eventBus.Stop(); err != nil {
		log.Error().Err(err).Msg("failed to stop event bus")
	}

	// 等待所有协程完成
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	// 等待完成或超时
	select {
	case <-done:
		// 正常完成不记录日志
	case <-time.After(30 * time.Second):
		log.Warn().Msg("timeout waiting for scheduler goroutines to stop")
	}

	// 撤销Leader租约
	if s.isLeader && s.leaderLease != 0 {
		if _, err := s.etcdClient.Client.Revoke(ctx, s.leaderLease); err != nil {
			log.Error().Err(err).Msg("failed to revoke leader lease")
		}
	}

	// 关闭ETCD会话
	if s.session != nil {
		if err := s.session.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close etcd session")
		}
	}

	s.isStarted = false

	log.Info().Str("nodeId", s.nodeID).Msg("scheduler stopped")
	return nil
}

// SubmitTask 提交任务（事件驱动）
func (s *scheduler) SubmitTask(ctx context.Context, task *Task) error {
	if !s.isStarted {
		return ErrSchedulerNotStarted
	}

	// 设置任务基本信息
	if task.ID == "" {
		task.ID = generateTaskID()
	}

	task.Status = TaskStatusPending
	task.CreatedAt = currentTimestamp()
	task.UpdatedAt = currentTimestamp()

	// 保存任务到ETCD
	if err := s.saveTaskToETCD(ctx, task); err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	return nil
}

// GetTask 获取任务信息
func (s *scheduler) GetTask(ctx context.Context, taskID string) (*Task, error) {
	return s.getTaskFromETCD(ctx, taskID)
}

// UpdateTask 更新任务信息
func (s *scheduler) UpdateTask(ctx context.Context, task *Task) error {
	task.UpdatedAt = currentTimestamp()
	return s.saveTaskToETCD(ctx, task)
}

// CancelTask 取消任务
func (s *scheduler) CancelTask(ctx context.Context, taskID string) error {
	task, err := s.getTaskFromETCD(ctx, taskID)
	if err != nil {
		return err
	}

	if task.Status == TaskStatusCompleted || task.Status == TaskStatusFailed {
		return fmt.Errorf("cannot cancel task in status: %s", task.Status.String())
	}

	task.Status = TaskStatusCanceled
	task.UpdatedAt = currentTimestamp()

	if err := s.saveTaskToETCD(ctx, task); err != nil {
		return err
	}

	// 发布任务取消事件
	s.publishEvent(&SchedulerEvent{
		Type:      EventTaskCanceled,
		TaskID:    task.ID,
		WorkerID:  task.WorkerID,
		Timestamp: currentTimestamp(),
	})

	log.Info().Str("taskId", taskID).Msg("task canceled successfully")
	return nil
}

// ListTasks 列出任务
func (s *scheduler) ListTasks(ctx context.Context, filter *TaskFilter) ([]*Task, error) {
	return s.listTasksFromETCD(ctx, filter)
}

// RegisterWorker 注册工作节点
func (s *scheduler) RegisterWorker(ctx context.Context, worker *Worker) error {
	// 设置基本信息
	if worker.ID == "" {
		worker.ID = generateWorkerID()
	}

	worker.RegisteredAt = currentTimestamp()
	worker.UpdatedAt = currentTimestamp()
	worker.LastHeartbeat = currentTimestamp()

	// 保存到ETCD
	if err := s.saveWorkerToETCD(ctx, worker); err != nil {
		return fmt.Errorf("failed to save worker: %w", err)
	}

	// 发布工作节点加入事件
	s.publishEvent(&SchedulerEvent{
		Type:      EventWorkerJoined,
		WorkerID:  worker.ID,
		Timestamp: currentTimestamp(),
		Data:      map[string]any{"worker": worker},
	})

	log.Info().
		Str("workerId", worker.ID).
		Str("workerName", worker.Name).
		Msg("worker registered successfully")

	return nil
}

// UnregisterWorker 注销工作节点
func (s *scheduler) UnregisterWorker(ctx context.Context, workerID string) error {
	// 如果是Leader，先重新调度该工作节点的任务
	if s.IsLeader() {
		if err := s.rescheduleWorkerTasks(ctx, workerID); err != nil {
			log.Error().Err(err).
				Str("workerId", workerID).
				Msg("failed to reschedule worker tasks before unregistering")
			// 继续删除节点，不因为重新调度失败而阻止注销
		}
	}

	// 删除工作节点
	if err := s.deleteWorkerFromETCD(ctx, workerID); err != nil {
		return fmt.Errorf("failed to delete worker: %w", err)
	}

	// 发布工作节点离开事件
	s.publishEvent(&SchedulerEvent{
		Type:      EventWorkerLeft,
		WorkerID:  workerID,
		Timestamp: currentTimestamp(),
	})

	log.Info().
		Str("workerId", workerID).
		Msg("worker unregistered successfully")

	return nil
}

// GetWorker 获取工作节点信息
func (s *scheduler) GetWorker(ctx context.Context, workerID string) (*Worker, error) {
	return s.getWorkerFromETCD(ctx, workerID)
}

// ListWorkers 列出工作节点
func (s *scheduler) ListWorkers(ctx context.Context, filter *WorkerFilter) ([]*Worker, error) {
	return s.listWorkersFromETCD(ctx, filter)
}

// RegisterEventCallback 注册事件回调
func (s *scheduler) RegisterEventCallback(eventType EventType, callback EventCallback) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.eventCallbacks[eventType] = append(s.eventCallbacks[eventType], callback)

	// 同时注册到事件总线
	s.eventBus.Subscribe(eventType, func(ctx context.Context, event *SchedulerEvent) error {
		callback(event)
		return nil
	})

	return nil
}

// IsLeader 检查当前节点是否为Leader
func (s *scheduler) IsLeader() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isLeader
}

// GetNodeID 获取当前节点ID
func (s *scheduler) GetNodeID() string {
	return s.nodeID
}

// GetMetrics 获取调度器指标
func (s *scheduler) GetMetrics() *SchedulerMetrics {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()

	return &SchedulerMetrics{
		TasksTotal:         s.metrics.TasksTotal,
		TasksPending:       s.metrics.TasksPending,
		TasksRunning:       s.metrics.TasksRunning,
		TasksCompleted:     s.metrics.TasksCompleted,
		TasksFailed:        s.metrics.TasksFailed,
		TasksCanceled:      s.metrics.TasksCanceled,
		WorkersTotal:       s.metrics.WorkersTotal,
		WorkersOnline:      s.metrics.WorkersOnline,
		WorkersOffline:     s.metrics.WorkersOffline,
		WorkersBusy:        s.metrics.WorkersBusy,
		WorkersIdle:        s.metrics.WorkersIdle,
		AvgTaskDuration:    s.metrics.AvgTaskDuration,
		TaskThroughput:     s.metrics.TaskThroughput,
		SchedulerUptime:    s.metrics.SchedulerUptime,
		LastScheduleTime:   s.metrics.LastScheduleTime,
		TotalScheduleCount: s.metrics.TotalScheduleCount,
		MemoryUsage:        s.metrics.MemoryUsage,
		GoroutineCount:     s.metrics.GoroutineCount,
		UpdatedAt:          s.metrics.UpdatedAt,
	}
}

// GetLoadBalanceStrategy 获取当前负载均衡策略
func (s *scheduler) GetLoadBalanceStrategy() LoadBalanceStrategy {
	return s.loadBalancer.GetStrategy()
}

// SetLoadBalanceStrategy 设置负载均衡策略
func (s *scheduler) SetLoadBalanceStrategy(strategy LoadBalanceStrategy) {
	s.loadBalancer.SetStrategy(strategy)
	log.Info().
		Str("nodeId", s.nodeID).
		Str("strategy", fmt.Sprintf("%v", strategy)).
		Msg("load balance strategy updated")
}

// ResetLoadBalancer 重置负载均衡器内部状态
func (s *scheduler) ResetLoadBalancer() {
	s.loadBalancer.Reset()
	log.Info().
		Str("nodeId", s.nodeID).
		Msg("load balancer reset")
}

// Health 健康检查
func (s *scheduler) Health(ctx context.Context) error {
	// 检查ETCD连接
	if _, err := s.etcdClient.Client.Status(ctx, s.etcdClient.Client.Endpoints()[0]); err != nil {
		return fmt.Errorf("etcd connection failed: %w", err)
	}

	// 检查调度器状态
	if !s.isStarted {
		return fmt.Errorf("scheduler not started")
	}

	return nil
}

// 以下是内部方法

// startLeaderElection 启动Leader选举
func (s *scheduler) startLeaderElection(ctx context.Context) error {
	electionKey := s.options.EtcdKeyPrefix + "election/"
	s.election = concurrency.NewElection(s.session, electionKey)

	s.wg.Add(1)
	go s.runLeaderElection(ctx)

	return nil
}

// runLeaderElection 运行Leader选举
func (s *scheduler) runLeaderElection(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		default:
			// 尝试成为Leader
			if err := s.election.Campaign(ctx, s.nodeID); err != nil {
				log.Error().Err(err).Msg("failed to campaign for leader")
				time.Sleep(5 * time.Second)
				continue
			}

			s.mu.Lock()
			s.isLeader = true
			s.mu.Unlock()

			// 发布Leader选举事件
			s.publishEvent(&SchedulerEvent{
				Type:      EventLeaderElected,
				Timestamp: currentTimestamp(),
				Data: map[string]any{
					"nodeId": s.nodeID,
				},
			})

			log.Info().
				Str("nodeId", s.nodeID).
				Msg("became leader")

			// 监听Leader状态
			select {
			case <-s.session.Done():
				s.mu.Lock()
				s.isLeader = false
				s.mu.Unlock()

				// 发布Leader丢失事件
				s.publishEvent(&SchedulerEvent{
					Type:      EventLeaderLost,
					Timestamp: currentTimestamp(),
					Data: map[string]any{
						"nodeId": s.nodeID,
					},
				})

				log.Warn().
					Str("nodeId", s.nodeID).
					Msg("lost leader status")
			case <-ctx.Done():
				return
			case <-s.stopChan:
				return
			}
		}
	}
}

// runETCDWatcher 运行ETCD监听器
func (s *scheduler) runETCDWatcher(ctx context.Context) {
	defer s.wg.Done()

	// 监听任务变化（包含之前的值用于状态比较）
	taskWatchCh := s.etcdClient.Client.Watch(ctx, s.options.EtcdKeyPrefix+"tasks/", clientv3.WithPrefix(), clientv3.WithPrevKV())

	// 监听工作节点变化
	workerWatchCh := s.etcdClient.Client.Watch(ctx, s.options.EtcdKeyPrefix+"workers/", clientv3.WithPrefix())

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case watchResp := <-taskWatchCh:
			s.handleTaskWatchEvents(ctx, watchResp)
		case watchResp := <-workerWatchCh:
			s.handleWorkerWatchEvents(ctx, watchResp)
		}
	}
}

// handleTaskWatchEvents 处理任务监听事件
func (s *scheduler) handleTaskWatchEvents(ctx context.Context, watchResp clientv3.WatchResponse) {
	for _, event := range watchResp.Events {
		switch event.Type {
		case clientv3.EventTypePut:
			// 检查是否是任务分配记录，如果是则跳过
			key := string(event.Kv.Key)
			if strings.Contains(key, "/assigned/") {
				continue
			}

			var task Task
			if err := json.Unmarshal(event.Kv.Value, &task); err != nil {
				log.Error().Err(err).Msg("failed to unmarshal task")
				continue
			}

			// 检查是否为真正的状态变化，避免重复事件
			if event.IsModify() && event.PrevKv != nil {
				var prevTask Task
				if err := json.Unmarshal(event.PrevKv.Value, &prevTask); err == nil {
					// 如果状态没有变化，跳过事件发布
					if prevTask.Status == task.Status {
						continue
					}
				}
			}

			// 根据任务状态发布不同事件
			var eventType EventType
			switch task.Status {
			case TaskStatusPending:
				eventType = EventTaskSubmitted
			case TaskStatusScheduled:
				eventType = EventTaskScheduled
			case TaskStatusRunning:
				eventType = EventTaskStarted
			case TaskStatusCompleted:
				eventType = EventTaskCompleted
			case TaskStatusFailed:
				eventType = EventTaskFailed
			case TaskStatusCanceled:
				eventType = EventTaskCanceled
			default:
				continue
			}

			s.publishEvent(&SchedulerEvent{
				Type:      eventType,
				TaskID:    task.ID,
				WorkerID:  task.WorkerID,
				Timestamp: currentTimestamp(),
				Data:      map[string]any{"task": &task},
			})

		case clientv3.EventTypeDelete:
			// 任务被删除
			log.Debug().Str("key", string(event.Kv.Key)).Msg("task deleted")
		}
	}
}

// handleWorkerWatchEvents 处理工作节点监听事件
func (s *scheduler) handleWorkerWatchEvents(ctx context.Context, watchResp clientv3.WatchResponse) {
	for _, event := range watchResp.Events {
		switch event.Type {
		case clientv3.EventTypePut:
			var worker Worker
			if err := json.Unmarshal(event.Kv.Value, &worker); err != nil {
				log.Error().Err(err).Msg("failed to unmarshal worker")
				continue
			}

			// 根据工作节点状态发布不同事件
			var eventType EventType
			switch worker.Status {
			case WorkerStatusOnline:
				eventType = EventWorkerOnline
			case WorkerStatusOffline:
				eventType = EventWorkerOffline
			case WorkerStatusBusy:
				eventType = EventWorkerBusy
			case WorkerStatusIdle:
				eventType = EventWorkerIdle
			default:
				eventType = EventWorkerJoined
			}

			s.publishEvent(&SchedulerEvent{
				Type:      eventType,
				WorkerID:  worker.ID,
				Timestamp: currentTimestamp(),
				Data:      map[string]any{"worker": &worker},
			})

		case clientv3.EventTypeDelete:
			// 工作节点被删除，需要重新调度其任务
			key := string(event.Kv.Key)
			log.Debug().Str("key", key).Msg("worker deleted")

			// 从key中提取workerID
			parts := strings.Split(key, "/")
			if len(parts) > 0 {
				workerID := parts[len(parts)-1]
				if workerID != "" {
					// 发布工作节点离开事件
					s.publishEvent(&SchedulerEvent{
						Type:      EventWorkerLeft,
						WorkerID:  workerID,
						Timestamp: currentTimestamp(),
					})
				}
			}
		}
	}
}

// runHealthCheck 运行健康检查
func (s *scheduler) runHealthCheck(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			if s.IsLeader() {
				s.checkWorkerHealth(ctx)
			}
		}
	}
}

// runPendingTask 运行待处理任务检查（安全网机制）
func (s *scheduler) runPendingTask(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(60 * time.Second) // 每60秒检查一次，频率较低
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			if s.IsLeader() {
				s.checkAndSchedulePendingTasks(ctx)
			}
		}
	}
}

// checkAndSchedulePendingTasks 检查并调度待处理任务（安全网）
func (s *scheduler) checkAndSchedulePendingTasks(ctx context.Context) {
	// 获取待调度任务（如果事件机制遗漏了某些任务）
	tasks, err := s.listTasksFromETCD(ctx, &TaskFilter{
		Status: []TaskStatus{TaskStatusPending},
		Limit:  50, // 限制数量
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to list pending tasks for safety net check")
		return
	}

	if len(tasks) == 0 {
		return
	}

	// 检查是否有可用工作节点
	workers, err := s.listWorkersFromETCD(ctx, &WorkerFilter{
		Status: []WorkerStatus{WorkerStatusOnline, WorkerStatusIdle},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to list workers for safety net check")
		return
	}

	if len(workers) == 0 {
		return
	}

	log.Info().
		Int("pendingTasks", len(tasks)).
		Int("availableWorkers", len(workers)).
		Msg("pending task safety net: found unscheduled tasks with available workers")

	// 调度这些任务
	for _, task := range tasks {
		if err := s.scheduleTask(ctx, task.ID); err != nil {
			log.Error().Err(err).
				Str("taskId", task.ID).
				Msg("failed to schedule pending task")
		}
	}
}

// runMetricsCollection 运行指标收集
func (s *scheduler) runMetricsCollection(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(60 * time.Second) // 每60秒收集一次
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.collectMetrics(ctx)
		}
	}
}

// assignTaskToWorker 分配任务给工作节点
func (s *scheduler) assignTaskToWorker(ctx context.Context, task *Task, worker *Worker) error {
	// 检查任务状态，避免重复分配
	if task.Status != TaskStatusPending {
		log.Debug().
			Str("taskId", task.ID).
			Str("status", task.Status.String()).
			Msg("task is not pending, skipping assignment")
		return nil
	}

	// 更新任务状态
	task.Status = TaskStatusScheduled
	task.WorkerID = worker.ID
	task.ScheduledAt = currentTimestamp()
	task.UpdatedAt = currentTimestamp()

	// 保存任务
	if err := s.saveTaskToETCD(ctx, task); err != nil {
		return fmt.Errorf("failed to save scheduled task: %w", err)
	}

	// 创建任务分配记录
	assignedKey := fmt.Sprintf("%stasks/assigned/%s/%s", s.options.EtcdKeyPrefix, worker.ID, task.ID)
	taskData, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	_, err = s.etcdClient.Client.Put(ctx, assignedKey, string(taskData))
	if err != nil {
		return fmt.Errorf("failed to create task assignment: %w", err)
	}

	return nil
}

// checkWorkerHealth 检查工作节点健康状态
func (s *scheduler) checkWorkerHealth(ctx context.Context) {
	workers, err := s.listWorkersFromETCD(ctx, &WorkerFilter{})
	if err != nil {
		log.Error().Err(err).Msg("failed to list workers for health check")
		return
	}

	now := currentTimestamp()
	timeoutMs := int64(s.options.WorkerTimeout.Milliseconds())

	for _, worker := range workers {
		// 检查工作节点是否超时
		if now-worker.LastHeartbeat > timeoutMs {
			if worker.Status != WorkerStatusOffline {
				log.Warn().
					Str("workerId", worker.ID).
					Int64("lastHeartbeat", worker.LastHeartbeat).
					Msg("worker timeout detected, marking as offline")

				// 标记工作节点为离线
				worker.Status = WorkerStatusOffline
				worker.UpdatedAt = now

				if err := s.saveWorkerToETCD(ctx, worker); err != nil {
					log.Error().Err(err).Str("workerId", worker.ID).Msg("failed to update worker status")
				}
			}
		}
	}
}

// collectMetrics 收集指标
func (s *scheduler) collectMetrics(ctx context.Context) {
	// 统计任务数量
	taskCounts := make(map[TaskStatus]int64)
	for status := TaskStatusPending; status <= TaskStatusRetrying; status++ {
		tasks, err := s.listTasksFromETCD(ctx, &TaskFilter{Status: []TaskStatus{status}})
		if err != nil {
			log.Error().Err(err).Msg("failed to count tasks for metrics")
			continue
		}
		taskCounts[status] = int64(len(tasks))
	}

	// 统计工作节点数量
	workerCounts := make(map[WorkerStatus]int64)
	for status := WorkerStatusOnline; status <= WorkerStatusIdle; status++ {
		workers, err := s.listWorkersFromETCD(ctx, &WorkerFilter{Status: []WorkerStatus{status}})
		if err != nil {
			log.Error().Err(err).Msg("failed to count workers for metrics")
			continue
		}
		workerCounts[status] = int64(len(workers))
	}

	// 更新指标
	s.updateMetrics(func(m *SchedulerMetrics) {
		m.TasksTotal = taskCounts[TaskStatusPending] + taskCounts[TaskStatusScheduled] +
			taskCounts[TaskStatusRunning] + taskCounts[TaskStatusCompleted] +
			taskCounts[TaskStatusFailed] + taskCounts[TaskStatusCanceled] + taskCounts[TaskStatusRetrying]
		m.TasksPending = taskCounts[TaskStatusPending]
		m.TasksRunning = taskCounts[TaskStatusRunning]
		m.TasksCompleted = taskCounts[TaskStatusCompleted]
		m.TasksFailed = taskCounts[TaskStatusFailed]
		m.TasksCanceled = taskCounts[TaskStatusCanceled]
		m.TasksRetrying = taskCounts[TaskStatusRetrying]

		m.WorkersTotal = workerCounts[WorkerStatusOnline] + workerCounts[WorkerStatusOffline] +
			workerCounts[WorkerStatusBusy] + workerCounts[WorkerStatusIdle]
		m.WorkersOnline = workerCounts[WorkerStatusOnline]
		m.WorkersOffline = workerCounts[WorkerStatusOffline]
		m.WorkersBusy = workerCounts[WorkerStatusBusy]
		m.WorkersIdle = workerCounts[WorkerStatusIdle]

		m.SchedulerUptime = time.Duration(currentTimestamp()-s.startTime) * time.Millisecond
		m.UpdatedAt = currentTimestamp()
	})
}

// 事件发布方法
func (s *scheduler) publishEvent(event *SchedulerEvent) {
	if err := s.eventBus.Publish(event); err != nil {
		log.Error().Err(err).Msg("failed to publish event")
	}
}

// 指标更新方法
func (s *scheduler) updateMetrics(updateFn func(*SchedulerMetrics)) {
	s.metrics.mu.Lock()
	updateFn(s.metrics)
	s.metrics.mu.Unlock()
}

// ETCD操作方法
func (s *scheduler) saveTaskToETCD(ctx context.Context, task *Task) error {
	key := fmt.Sprintf("%stasks/%s", s.options.EtcdKeyPrefix, task.ID)
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}
	_, err = s.etcdClient.Client.Put(ctx, key, string(data))
	return err
}

func (s *scheduler) getTaskFromETCD(ctx context.Context, taskID string) (*Task, error) {
	key := fmt.Sprintf("%stasks/%s", s.options.EtcdKeyPrefix, taskID)
	resp, err := s.etcdClient.Client.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, ErrTaskNotFound
	}

	var task Task
	if err := json.Unmarshal(resp.Kvs[0].Value, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *scheduler) listTasksFromETCD(ctx context.Context, filter *TaskFilter) ([]*Task, error) {
	key := fmt.Sprintf("%stasks/", s.options.EtcdKeyPrefix)
	resp, err := s.etcdClient.Client.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var tasks []*Task
	for _, kv := range resp.Kvs {
		var task Task
		if err := json.Unmarshal(kv.Value, &task); err != nil {
			continue
		}

		// 应用过滤器
		if filter != nil {
			if len(filter.Status) > 0 {
				found := false
				for _, status := range filter.Status {
					if task.Status == status {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			if filter.Limit > 0 && len(tasks) >= filter.Limit {
				break
			}
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}

func (s *scheduler) saveWorkerToETCD(ctx context.Context, worker *Worker) error {
	key := fmt.Sprintf("%sworkers/%s", s.options.EtcdKeyPrefix, worker.ID)
	data, err := json.Marshal(worker)
	if err != nil {
		return err
	}
	_, err = s.etcdClient.Client.Put(ctx, key, string(data))
	return err
}

func (s *scheduler) getWorkerFromETCD(ctx context.Context, workerID string) (*Worker, error) {
	key := fmt.Sprintf("%sworkers/%s", s.options.EtcdKeyPrefix, workerID)
	resp, err := s.etcdClient.Client.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, ErrWorkerNotFound
	}

	var worker Worker
	if err := json.Unmarshal(resp.Kvs[0].Value, &worker); err != nil {
		return nil, err
	}
	return &worker, nil
}

func (s *scheduler) deleteWorkerFromETCD(ctx context.Context, workerID string) error {
	key := fmt.Sprintf("%sworkers/%s", s.options.EtcdKeyPrefix, workerID)
	_, err := s.etcdClient.Client.Delete(ctx, key)
	return err
}

func (s *scheduler) listWorkersFromETCD(ctx context.Context, filter *WorkerFilter) ([]*Worker, error) {
	key := fmt.Sprintf("%sworkers/", s.options.EtcdKeyPrefix)
	resp, err := s.etcdClient.Client.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var workers []*Worker
	for _, kv := range resp.Kvs {
		var worker Worker
		if err := json.Unmarshal(kv.Value, &worker); err != nil {
			continue
		}

		// 应用过滤器
		if filter != nil {
			if len(filter.Status) > 0 {
				found := false
				for _, status := range filter.Status {
					if worker.Status == status {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			if filter.Limit > 0 && len(workers) >= filter.Limit {
				break
			}
		}

		workers = append(workers, &worker)
	}

	return workers, nil
}

// 任务调度相关方法（事件驱动实现）
func (s *scheduler) scheduleTask(ctx context.Context, taskID string) error {
	task, err := s.getTaskFromETCD(ctx, taskID)
	if err != nil {
		return err
	}

	if task.Status != TaskStatusPending {
		return nil // 任务已经被调度
	}

	// 获取可用工作节点
	workers, err := s.listWorkersFromETCD(ctx, &WorkerFilter{
		Status: []WorkerStatus{WorkerStatusOnline, WorkerStatusIdle},
	})
	if err != nil {
		return err
	}

	if len(workers) == 0 {
		log.Warn().Str("taskId", taskID).Msg("no available workers for task")
		return nil
	}

	// 使用负载均衡策略选择工作节点
	worker, err := s.selectBestWorker(ctx, workers, task)
	if err != nil {
		return fmt.Errorf("failed to select worker: %w", err)
	}

	return s.assignTaskToWorker(ctx, task, worker)
}

// schedulePendingTasksForNewWorker 当新工作节点加入时调度待处理任务
func (s *scheduler) schedulePendingTasksForNewWorker(ctx context.Context) {
	if !s.IsLeader() {
		return
	}

	// 获取待调度任务（限制数量，避免一次处理过多）
	tasks, err := s.listTasksFromETCD(ctx, &TaskFilter{
		Status: []TaskStatus{TaskStatusPending},
		Limit:  10, // 每次最多处理10个任务
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to list pending tasks for new worker")
		return
	}

	if len(tasks) == 0 {
		return
	}

	// 获取可用工作节点
	workers, err := s.listWorkersFromETCD(ctx, &WorkerFilter{
		Status: []WorkerStatus{WorkerStatusOnline, WorkerStatusIdle},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to list available workers")
		return
	}

	if len(workers) == 0 {
		return
	}

	// 为可用的工作节点调度任务
	scheduledCount := 0
	for _, task := range tasks {
		// 使用负载均衡策略选择最佳工作节点
		worker, err := s.selectBestWorker(ctx, workers, task)
		if err != nil {
			log.Error().Err(err).
				Str("taskId", task.ID).
				Msg("failed to select worker for task")
			continue
		}

		if err := s.assignTaskToWorker(ctx, task, worker); err != nil {
			log.Error().Err(err).
				Str("taskId", task.ID).
				Str("workerId", worker.ID).
				Msg("failed to assign task to worker")
		} else {
			scheduledCount++
		}
	}

	if scheduledCount > 0 {
		log.Info().
			Int("scheduledCount", scheduledCount).
			Msg("scheduled pending tasks for available workers")
	}
}

func (s *scheduler) rescheduleWorkerTasks(ctx context.Context, workerID string) error {
	// 获取该工作节点的所有任务
	tasks, err := s.listTasksFromETCD(ctx, &TaskFilter{})
	if err != nil {
		return err
	}

	rescheduledCount := 0
	// 重新调度该工作节点的运行中和已调度任务
	for _, task := range tasks {
		if task.WorkerID == workerID && (task.Status == TaskStatusRunning || task.Status == TaskStatusScheduled) {
			oldStatus := task.Status
			task.Status = TaskStatusPending
			task.WorkerID = ""
			task.ScheduledAt = 0 // 清除调度时间
			task.UpdatedAt = currentTimestamp()

			if err := s.saveTaskToETCD(ctx, task); err != nil {
				log.Error().Err(err).
					Str("taskId", task.ID).
					Str("oldStatus", oldStatus.String()).
					Msg("failed to reschedule task")
			} else {
				rescheduledCount++
				log.Info().
					Str("taskId", task.ID).
					Str("oldStatus", oldStatus.String()).
					Str("workerId", workerID).
					Msg("task rescheduled due to worker offline")
			}
		}
	}

	// 清理任务分配记录
	assignedPrefix := fmt.Sprintf("%stasks/assigned/%s/", s.options.EtcdKeyPrefix, workerID)
	if _, err := s.etcdClient.Client.Delete(ctx, assignedPrefix, clientv3.WithPrefix()); err != nil {
		log.Error().Err(err).
			Str("workerId", workerID).
			Msg("failed to clean up task assignments")
	}

	if rescheduledCount > 0 {
		log.Info().
			Str("workerId", workerID).
			Int("rescheduledCount", rescheduledCount).
			Msg("completed rescheduling tasks for offline worker")
	}

	return nil
}

// selectBestWorker 根据负载均衡策略选择最佳工作节点
func (s *scheduler) selectBestWorker(ctx context.Context, workers []*Worker, task *Task) (*Worker, error) {
	return s.loadBalancer.SelectWorker(ctx, workers, task)
}

// getWorkerRunningTaskCount 获取工作节点的运行中任务数
func (s *scheduler) getWorkerRunningTaskCount(ctx context.Context, workerID string) (int, error) {
	// 获取分配给该工作节点的任务
	prefix := fmt.Sprintf("%stasks/assigned/%s/", s.options.EtcdKeyPrefix, workerID)
	resp, err := s.etcdClient.Client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return 0, err
	}

	// 统计运行中的任务
	runningTasks := 0
	for _, kv := range resp.Kvs {
		var taskData map[string]any
		if err := json.Unmarshal(kv.Value, &taskData); err != nil {
			continue
		}

		if status, ok := taskData["status"].(string); ok {
			if status == TaskStatusRunning.String() || status == TaskStatusScheduled.String() {
				runningTasks++
			}
		}
	}

	return runningTasks, nil
}
