package scheduler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/store/redis"
	redisClient "github.com/redis/go-redis/v9"
)

// redisScheduler Redis分布式任务调度器实现
type redisScheduler struct {
	// 基本信息
	nodeID  string
	options *SchedulerOptions

	// Redis客户端
	redisClient redisClient.UniversalClient
	keys        *RedisKeys

	// 核心组件
	taskManager   *TaskManager
	workerManager *WorkerManager
	eventManager  *EventManager
	loadBalancer  LoadBalancer
	leaderElector LeaderElector
	lockManager   *RedisLockManager

	// 状态管理
	mu        sync.RWMutex
	isStarted bool
	isLeader  bool

	// 事件回调
	eventCallbacks map[EventType][]EventCallback

	// 指标
	metrics   *SchedulerMetrics
	startTime int64

	// 控制通道
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewScheduler 创建Redis分布式调度器
func NewScheduler(redisConfig *redis.SingleConfig, options *SchedulerOptions) (Scheduler, error) {
	if options == nil {
		options = DefaultSchedulerOptions()
	}

	// 生成节点ID
	if options.NodeID == "" {
		options.NodeID = generateNodeID()
	}

	// 创建Redis客户端
	client, err := redis.NewClient(redisConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis client: %w", err)
	}

	// 创建Redis键管理
	keys := NewRedisKeys("scheduler")

	// 创建调度器实例
	scheduler := &redisScheduler{
		nodeID:         options.NodeID,
		options:        options,
		redisClient:    client.Client,
		keys:           keys,
		eventCallbacks: make(map[EventType][]EventCallback),
		stopChan:       make(chan struct{}),
		metrics:        &SchedulerMetrics{NodeID: options.NodeID},
	}

	// 初始化组件
	scheduler.taskManager = NewTaskManager(client.Client, keys)
	scheduler.workerManager = NewWorkerManager(client.Client, keys, options.WorkerLeaseTTL)
	scheduler.eventManager = NewEventManager(client.Client, keys, options.StreamGroup, options.StreamConsumerID, options.StreamMaxLen)
	scheduler.lockManager = NewRedisLockManager(client.Client)
	scheduler.leaderElector = NewRedisLeaderElector(client.Client, options.NodeID, keys.LeaderLock, options.LeaderLeaseDuration)

	// 初始化负载均衡器
	countProvider := NewRedisTaskCountProvider(scheduler.workerManager)
	scheduler.loadBalancer = NewRedisLoadBalancer(options.LoadBalanceStrategy, countProvider)

	return scheduler, nil
}

// Start 启动调度器
func (s *redisScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isStarted {
		return ErrSchedulerAlreadyStarted
	}

	s.startTime = time.Now().Unix()

	log.Info().Str("nodeId", s.nodeID).Msg("starting redis scheduler")

	// 启动事件管理器
	if err := s.eventManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start event manager: %w", err)
	}

	// 注册核心事件回调
	s.registerCoreEventCallbacks()

	// 启动Leader选举
	if err := s.leaderElector.CampaignLeader(ctx); err != nil {
		log.Error().Err(err).Msg("failed to campaign for leader, will retry")
	}

	// 启动主循环
	s.wg.Add(1)
	go s.runMainLoop(ctx)

	// 启动Leader选举监控
	s.wg.Add(1)
	go s.runLeaderElectionMonitor(ctx)

	// 启动孤儿任务检测和恢复
	s.wg.Add(1)
	go s.runOrphanTaskRecovery(ctx)

	// 启动健康检查
	if s.options.HealthCheckInterval > 0 {
		s.wg.Add(1)
		go s.runHealthCheck(ctx)
	}

	// 启动指标收集
	if s.options.EnableMetrics {
		s.wg.Add(1)
		go s.runMetricsCollection(ctx)
	}

	s.isStarted = true

	// 发布调度器启动事件
	s.publishEvent(ctx, &SchedulerEvent{
		Type:      EventSchedulerStarted,
		NodeID:    s.nodeID,
		Timestamp: time.Now().Unix(),
		Data: map[string]any{
			"nodeId": s.nodeID,
		},
	})

	log.Info().Str("nodeId", s.nodeID).Msg("redis scheduler started successfully")
	return nil
}

// Stop 停止调度器
func (s *redisScheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isStarted {
		return ErrSchedulerNotStarted
	}

	log.Info().Str("nodeId", s.nodeID).Msg("stopping redis scheduler")

	// 发布调度器停止事件
	s.publishEvent(ctx, &SchedulerEvent{
		Type:      EventSchedulerStopped,
		NodeID:    s.nodeID,
		Timestamp: time.Now().Unix(),
		Data: map[string]any{
			"nodeId": s.nodeID,
		},
	})

	// 停止接收新任务
	close(s.stopChan)

	// 主动放弃Leader
	if s.isLeader {
		if err := s.leaderElector.ResignLeader(ctx); err != nil {
			log.Error().Err(err).Msg("failed to resign leader")
		}
	}

	// 等待所有协程完成
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		log.Warn().Msg("timeout waiting for scheduler goroutines to stop")
	}

	// 停止组件
	s.workerManager.Stop()
	if err := s.eventManager.Stop(); err != nil {
		log.Error().Err(err).Msg("failed to stop event manager")
	}
	if err := s.lockManager.Close(ctx); err != nil {
		log.Error().Err(err).Msg("failed to close lock manager")
	}

	s.isStarted = false
	s.isLeader = false

	log.Info().Str("nodeId", s.nodeID).Msg("redis scheduler stopped")
	return nil
}

// SubmitTask 提交任务
func (s *redisScheduler) SubmitTask(ctx context.Context, task *Task) error {
	if !s.isStarted {
		return ErrSchedulerNotStarted
	}

	// 设置任务基本信息
	if task.ID == "" {
		task.ID = generateTaskID()
	}

	task.Status = TaskStatusPending
	task.CreatedAt = time.Now().Unix()
	task.UpdatedAt = task.CreatedAt
	task.Version = 1

	// 保存任务
	if err := s.taskManager.SaveTask(ctx, task); err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	// 发布任务提交事件
	if err := s.eventManager.PublishTaskEvent(ctx, EventTaskSubmitted, task, ""); err != nil {
		log.Error().Err(err).Str("taskId", task.ID).Msg("failed to publish task submitted event")
	}

	log.Info().Str("taskId", task.ID).Str("priority", task.Priority.String()).Msg("task submitted")
	return nil
}

// GetTask 获取任务信息
func (s *redisScheduler) GetTask(ctx context.Context, taskID string) (*Task, error) {
	return s.taskManager.GetTask(ctx, taskID)
}

// UpdateTask 更新任务信息
func (s *redisScheduler) UpdateTask(ctx context.Context, task *Task) error {
	task.UpdatedAt = time.Now().Unix()
	task.Version++
	return s.taskManager.SaveTask(ctx, task)
}

// CancelTask 取消任务
func (s *redisScheduler) CancelTask(ctx context.Context, taskID string) error {
	task, err := s.taskManager.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	if task.Status == TaskStatusCompleted || task.Status == TaskStatusFailed {
		return fmt.Errorf("cannot cancel task in status: %s", task.Status.String())
	}

	// 更新任务状态
	if err := s.taskManager.UpdateTaskStatus(ctx, taskID, TaskStatusCanceled); err != nil {
		return err
	}

	// 发布任务取消事件
	if err := s.eventManager.PublishTaskEvent(ctx, EventTaskCanceled, task, ""); err != nil {
		log.Error().Err(err).Str("taskId", taskID).Msg("failed to publish task canceled event")
	}

	log.Info().Str("taskId", taskID).Msg("task canceled successfully")
	return nil
}

// ListTasks 列出任务
func (s *redisScheduler) ListTasks(ctx context.Context, filter *TaskFilter) ([]*Task, error) {
	return s.taskManager.ListTasks(ctx, filter)
}

// RegisterWorker 注册工作节点
func (s *redisScheduler) RegisterWorker(ctx context.Context, worker *Worker) error {
	if err := s.workerManager.RegisterWorker(ctx, worker); err != nil {
		return err
	}

	// 发布工作节点加入事件
	if err := s.eventManager.PublishWorkerEvent(ctx, EventWorkerJoined, worker); err != nil {
		log.Error().Err(err).Str("workerId", worker.ID).Msg("failed to publish worker joined event")
	}

	return nil
}

// UnregisterWorker 注销工作节点
func (s *redisScheduler) UnregisterWorker(ctx context.Context, workerID string) error {
	worker, err := s.workerManager.GetWorker(ctx, workerID)
	if err != nil {
		return err
	}

	if err := s.workerManager.UnregisterWorker(ctx, workerID); err != nil {
		return err
	}

	// 发布工作节点离开事件
	if err := s.eventManager.PublishWorkerEvent(ctx, EventWorkerLeft, worker); err != nil {
		log.Error().Err(err).Str("workerId", workerID).Msg("failed to publish worker left event")
	}

	return nil
}

// GetWorker 获取工作节点信息
func (s *redisScheduler) GetWorker(ctx context.Context, workerID string) (*Worker, error) {
	return s.workerManager.GetWorker(ctx, workerID)
}

// ListWorkers 列出工作节点
func (s *redisScheduler) ListWorkers(ctx context.Context, filter *WorkerFilter) ([]*Worker, error) {
	return s.workerManager.ListWorkers(ctx, filter)
}

// RegisterEventCallback 注册事件回调
func (s *redisScheduler) RegisterEventCallback(eventType EventType, callback EventCallback) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.eventCallbacks[eventType] = append(s.eventCallbacks[eventType], callback)
	s.eventManager.RegisterCallback(eventType, callback)

	return nil
}

// IsLeader 检查是否为Leader
func (s *redisScheduler) IsLeader() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isLeader
}

// GetNodeID 获取节点ID
func (s *redisScheduler) GetNodeID() string {
	return s.nodeID
}

// GetMetrics 获取调度器指标
func (s *redisScheduler) GetMetrics() *SchedulerMetrics {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()

	// 创建新的指标对象以避免复制锁
	metrics := &SchedulerMetrics{
		NodeID:           s.metrics.NodeID,
		IsLeader:         s.isLeader,
		StartTime:        s.startTime,
		Uptime:           time.Now().Unix() - s.startTime,
		LastLeaderChange: s.metrics.LastLeaderChange,
		LeaderTerm:       s.metrics.LeaderTerm,
		TasksSubmitted:   s.metrics.TasksSubmitted,
		TasksScheduled:   s.metrics.TasksScheduled,
		TasksCompleted:   s.metrics.TasksCompleted,
		TasksFailed:      s.metrics.TasksFailed,
		TasksCanceled:    s.metrics.TasksCanceled,
		TasksRetrying:    s.metrics.TasksRetrying,
		PendingTasks:     s.metrics.PendingTasks,
		RunningTasks:     s.metrics.RunningTasks,
		OnlineWorkers:    s.metrics.OnlineWorkers,
		OfflineWorkers:   s.metrics.OfflineWorkers,
		BusyWorkers:      s.metrics.BusyWorkers,
		IdleWorkers:      s.metrics.IdleWorkers,
	}

	return metrics
}

// GetLoadBalanceStrategy 获取负载均衡策略
func (s *redisScheduler) GetLoadBalanceStrategy() LoadBalanceStrategy {
	return s.loadBalancer.GetStrategy()
}

// SetLoadBalanceStrategy 设置负载均衡策略
func (s *redisScheduler) SetLoadBalanceStrategy(strategy LoadBalanceStrategy) {
	s.loadBalancer.SetStrategy(strategy)
}

// ResetLoadBalancer 重置负载均衡器
func (s *redisScheduler) ResetLoadBalancer() {
	s.loadBalancer.Reset()
}

// Health 健康检查
func (s *redisScheduler) Health(ctx context.Context) error {
	// 检查Redis连接
	if err := s.redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis connection failed: %w", err)
	}

	// 检查组件状态
	if !s.isStarted {
		return fmt.Errorf("scheduler not started")
	}

	return nil
}

// runMainLoop 运行主循环
func (s *redisScheduler) runMainLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Second) // 每秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			// 只有Leader才执行任务调度
			if s.isLeader {
				if err := s.scheduleNextTask(ctx); err != nil {
					log.Error().Err(err).Msg("failed to schedule task")
				}
			}
		}
	}
}

// scheduleNextTask 调度下一个任务
func (s *redisScheduler) scheduleNextTask(ctx context.Context) error {
	// 从队列中获取待处理任务
	task, err := s.taskManager.PopTask(ctx)
	if err != nil {
		return fmt.Errorf("failed to pop task: %w", err)
	}

	if task == nil {
		return nil // 没有待处理任务
	}

	// 获取可用工作节点
	workers, err := s.workerManager.GetAvailableWorkers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get available workers: %w", err)
	}

	if len(workers) == 0 {
		// 没有可用工作节点，将任务重新放回队列
		task.Status = TaskStatusPending
		if err := s.taskManager.SaveTask(ctx, task); err != nil {
			log.Error().Err(err).Str("taskId", task.ID).Msg("failed to save task back to queue")
		}
		return nil
	}

	// 使用负载均衡器选择工作节点
	selectedWorker, err := s.loadBalancer.SelectWorker(ctx, workers, task)
	if err != nil {
		return fmt.Errorf("failed to select worker: %w", err)
	}

	// 分配任务给工作节点
	task.WorkerID = selectedWorker.ID
	task.Status = TaskStatusScheduled
	task.ScheduledAt = time.Now().Unix()
	task.UpdatedAt = task.ScheduledAt

	if err := s.taskManager.SaveTask(ctx, task); err != nil {
		return fmt.Errorf("failed to save scheduled task: %w", err)
	}

	// 更新工作节点任务数
	if err := s.workerManager.UpdateWorkerTaskCount(ctx, selectedWorker.ID, selectedWorker.CurrentTasks+1); err != nil {
		log.Error().Err(err).Str("workerId", selectedWorker.ID).Msg("failed to update worker task count")
	}

	// 发布任务调度事件
	if err := s.eventManager.PublishTaskEvent(ctx, EventTaskScheduled, task, ""); err != nil {
		log.Error().Err(err).Str("taskId", task.ID).Msg("failed to publish task scheduled event")
	}

	log.Info().
		Str("taskId", task.ID).
		Str("workerId", selectedWorker.ID).
		Int8("strategy", int8(s.loadBalancer.GetStrategy())).
		Msg("task scheduled successfully")

	return nil
}

// runLeaderElectionMonitor 运行Leader选举监控
func (s *redisScheduler) runLeaderElectionMonitor(ctx context.Context) {
	defer s.wg.Done()

	eventChan := s.leaderElector.WatchLeaderElection(ctx)
	ticker := time.NewTicker(30 * time.Second) // 定期尝试竞选
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case event := <-eventChan:
			s.handleLeaderElectionEvent(ctx, event)
		case <-ticker.C:
			if !s.isLeader {
				// 如果不是Leader，尝试竞选
				if err := s.leaderElector.CampaignLeader(ctx); err != nil {
					log.Debug().Err(err).Msg("failed to campaign for leader")
				}
			}
		}
	}
}

// handleLeaderElectionEvent 处理Leader选举事件
func (s *redisScheduler) handleLeaderElectionEvent(ctx context.Context, event LeaderElectionEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch event.Type {
	case "elected":
		if event.NodeID == s.nodeID {
			s.isLeader = true
			s.metrics.LastLeaderChange = time.Now().Unix()
			s.metrics.LeaderTerm++

			log.Info().Str("nodeId", s.nodeID).Msg("became leader")

			// 发布Leader选举事件
			s.publishEvent(ctx, &SchedulerEvent{
				Type:      EventLeaderElected,
				NodeID:    s.nodeID,
				Timestamp: time.Now().Unix(),
				Data: map[string]any{
					"nodeId": s.nodeID,
					"term":   s.metrics.LeaderTerm,
				},
			})
		}
	case "lost":
		if event.NodeID == s.nodeID {
			s.isLeader = false
			s.metrics.LastLeaderChange = time.Now().Unix()

			log.Info().Str("nodeId", s.nodeID).Msg("lost leadership")

			// 发布Leader丢失事件
			s.publishEvent(ctx, &SchedulerEvent{
				Type:      EventLeaderLost,
				NodeID:    s.nodeID,
				Timestamp: time.Now().Unix(),
				Data: map[string]any{
					"nodeId": s.nodeID,
				},
			})
		}
	}
}

// runHealthCheck 运行健康检查
func (s *redisScheduler) runHealthCheck(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.options.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			if err := s.Health(ctx); err != nil {
				log.Error().Err(err).Msg("health check failed")
			}
		}
	}
}

// runMetricsCollection 运行指标收集
func (s *redisScheduler) runMetricsCollection(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.options.MetricsInterval)
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

// collectMetrics 收集指标
func (s *redisScheduler) collectMetrics(ctx context.Context) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()

	// 收集任务指标
	if pendingCount, err := s.taskManager.GetPendingTaskCount(ctx); err == nil {
		s.metrics.PendingTasks = pendingCount
	}

	if runningCount, err := s.taskManager.GetRunningTaskCount(ctx); err == nil {
		s.metrics.RunningTasks = runningCount
	}

	if completedCount, err := s.taskManager.GetCompletedTaskCount(ctx); err == nil {
		s.metrics.TasksCompleted = completedCount
	}

	if failedCount, err := s.taskManager.GetFailedTaskCount(ctx); err == nil {
		s.metrics.TasksFailed = failedCount
	}

	// 收集工作节点指标
	if workerCounts, err := s.workerManager.GetWorkerCount(ctx); err == nil {
		s.metrics.OnlineWorkers = workerCounts[WorkerStatusOnline]
		s.metrics.BusyWorkers = workerCounts[WorkerStatusBusy]
		s.metrics.OfflineWorkers = workerCounts[WorkerStatusOffline]
		s.metrics.IdleWorkers = s.metrics.OnlineWorkers - s.metrics.BusyWorkers
	}
}

// registerCoreEventCallbacks 注册核心事件回调
func (s *redisScheduler) registerCoreEventCallbacks() {
	// 任务完成事件
	s.eventManager.RegisterCallback(EventTaskCompleted, func(ctx context.Context, event *SchedulerEvent) error {
		s.metrics.mu.Lock()
		s.metrics.TasksCompleted++
		s.metrics.mu.Unlock()

		// 更新工作节点任务数
		if event.WorkerID != "" {
			if worker, err := s.workerManager.GetWorker(ctx, event.WorkerID); err == nil {
				newCount := worker.CurrentTasks - 1
				if newCount < 0 {
					newCount = 0
				}
				if err := s.workerManager.UpdateWorkerTaskCount(ctx, event.WorkerID, newCount); err != nil {
					log.Error().Err(err).Str("workerId", event.WorkerID).Msg("failed to update worker task count")
				}
			}
		}

		return nil
	})

	// 任务失败事件
	s.eventManager.RegisterCallback(EventTaskFailed, func(ctx context.Context, event *SchedulerEvent) error {
		s.metrics.mu.Lock()
		s.metrics.TasksFailed++
		s.metrics.mu.Unlock()

		// 更新工作节点任务数
		if event.WorkerID != "" {
			if worker, err := s.workerManager.GetWorker(ctx, event.WorkerID); err == nil {
				newCount := worker.CurrentTasks - 1
				if newCount < 0 {
					newCount = 0
				}
				if err := s.workerManager.UpdateWorkerTaskCount(ctx, event.WorkerID, newCount); err != nil {
					log.Error().Err(err).Str("workerId", event.WorkerID).Msg("failed to update worker task count")
				}
			}
		}

		return nil
	})
}

// publishEvent 发布事件的便捷方法
func (s *redisScheduler) publishEvent(ctx context.Context, event *SchedulerEvent) {
	if err := s.eventManager.PublishEvent(ctx, event); err != nil {
		log.Error().Err(err).Str("eventType", string(event.Type)).Msg("failed to publish event")
	}
}

// generateNodeID 生成节点ID
func generateNodeID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("node-%d", time.Now().UnixNano())
	}
	return "node-" + hex.EncodeToString(bytes)
}

// generateTaskID 生成任务ID
func generateTaskID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("task-%d", time.Now().UnixNano())
	}
	return "task-" + hex.EncodeToString(bytes)
}

// runOrphanTaskRecovery 运行孤儿任务恢复循环
func (s *redisScheduler) runOrphanTaskRecovery(ctx context.Context) {
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
			// 只有Leader才执行孤儿任务恢复
			if s.isLeader {
				if err := s.detectAndRecoverOrphanTasks(ctx); err != nil {
					log.Error().Err(err).Msg("failed to recover orphan tasks")
				}
			}
		}
	}
}

// detectAndRecoverOrphanTasks 检测并恢复孤儿任务
func (s *redisScheduler) detectAndRecoverOrphanTasks(ctx context.Context) error {
	// 获取所有工作节点
	allWorkers, err := s.workerManager.ListWorkers(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list all workers: %w", err)
	}

	// 获取在线工作节点
	onlineWorkers, err := s.workerManager.GetOnlineWorkers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get online workers: %w", err)
	}

	// 创建在线工作节点映射
	onlineWorkerMap := make(map[string]bool)
	for _, worker := range onlineWorkers {
		onlineWorkerMap[worker.ID] = true
	}

	// 找出下线的工作节点
	var offlineWorkerIDs []string
	for _, worker := range allWorkers {
		if !onlineWorkerMap[worker.ID] {
			offlineWorkerIDs = append(offlineWorkerIDs, worker.ID)
		}
	}

	if len(offlineWorkerIDs) == 0 {
		return nil // 没有下线的工作节点
	}

	// 查找孤儿任务
	orphanTasks, err := s.taskManager.FindOrphanTasks(ctx, offlineWorkerIDs)
	if err != nil {
		return fmt.Errorf("failed to find orphan tasks: %w", err)
	}

	if len(orphanTasks) == 0 {
		return nil // 没有孤儿任务
	}

	log.Info().
		Int("orphanCount", len(orphanTasks)).
		Strs("offlineWorkers", offlineWorkerIDs).
		Msg("found orphan tasks, starting recovery")

	// 重调度孤儿任务
	if err := s.taskManager.RescheduleOrphanTasks(ctx, orphanTasks); err != nil {
		return fmt.Errorf("failed to reschedule orphan tasks: %w", err)
	}

	// 发布任务重调度事件
	for _, task := range orphanTasks {
		if err := s.eventManager.PublishTaskEvent(ctx, EventTaskRescheduled, task, "Worker offline recovery"); err != nil {
			log.Error().Err(err).Str("taskId", task.ID).Msg("failed to publish task rescheduled event")
		}
	}

	log.Info().
		Int("rescheduledCount", len(orphanTasks)).
		Msg("orphan tasks recovery completed")

	return nil
}

// handleWorkerOffline 处理工作节点下线
func (s *redisScheduler) handleWorkerOffline(ctx context.Context, workerID string) error {
	log.Info().Str("workerId", workerID).Msg("handling worker offline")

	// 获取工作节点信息
	worker, err := s.workerManager.GetWorker(ctx, workerID)
	if err != nil && err != ErrWorkerNotFound {
		return fmt.Errorf("failed to get worker info: %w", err)
	}

	// 如果工作节点已经不存在，直接返回
	if err == ErrWorkerNotFound {
		log.Warn().Str("workerId", workerID).Msg("worker not found, may already be removed")
		return nil
	}

	// 发布工作节点下线事件
	if err := s.eventManager.PublishWorkerEvent(ctx, EventWorkerOffline, worker); err != nil {
		log.Error().Err(err).Str("workerId", workerID).Msg("failed to publish worker offline event")
	}

	// 如果是Leader，立即检测并恢复该工作节点的任务
	if s.isLeader {
		orphanTasks, err := s.taskManager.FindOrphanTasks(ctx, []string{workerID})
		if err != nil {
			log.Error().Err(err).Str("workerId", workerID).Msg("failed to find orphan tasks for offline worker")
		} else if len(orphanTasks) > 0 {
			log.Info().
				Int("orphanCount", len(orphanTasks)).
				Str("workerId", workerID).
				Msg("found orphan tasks for offline worker, starting immediate recovery")

			if err := s.taskManager.RescheduleOrphanTasks(ctx, orphanTasks); err != nil {
				log.Error().Err(err).Str("workerId", workerID).Msg("failed to reschedule orphan tasks")
			} else {
				// 发布任务重调度事件
				for _, task := range orphanTasks {
					if err := s.eventManager.PublishTaskEvent(ctx, EventTaskRescheduled, task, "Worker offline recovery"); err != nil {
						log.Error().Err(err).Str("taskId", task.ID).Msg("failed to publish task rescheduled event")
					}
				}
				log.Info().
					Int("rescheduledCount", len(orphanTasks)).
					Str("workerId", workerID).
					Msg("immediate orphan task recovery completed")
			}
		}
	}

	return nil
}
