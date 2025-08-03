package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/store/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// WorkerNode 事件驱动的工作节点
type WorkerNode struct {
	// 基本信息
	id      string
	name    string
	options *WorkerOptions

	// ETCD组件
	etcdClient *etcd.Etcd

	// 任务处理
	processor TaskProcessor
	taskChan  chan *Task

	// 状态管理
	mu        sync.RWMutex
	isStarted bool
	status    WorkerStatus

	// 租约管理
	leaseID clientv3.LeaseID

	// 控制通道
	stopChan chan struct{}
	wg       sync.WaitGroup

	// 指标
	metrics *WorkerMetrics
}

// WorkerMetrics 工作节点指标
type WorkerMetrics struct {
	TasksProcessed int64         `json:"tasksProcessed"`
	TasksFailed    int64         `json:"tasksFailed"`
	AvgDuration    time.Duration `json:"avgDuration"`
	LastTaskTime   int64         `json:"lastTaskTime"`
	mu             sync.RWMutex  `json:"-"`
}

// NewWorkerNode 创建工作节点实例
func NewWorkerNode(etcdClient *etcd.Etcd, options *WorkerOptions) (*WorkerNode, error) {
	if options == nil {
		options = DefaultWorkerOptions()
	}

	if options.WorkerID == "" {
		options.WorkerID = generateWorkerID()
	}

	if options.WorkerName == "" {
		options.WorkerName = options.WorkerID
	}

	worker := &WorkerNode{
		id:         options.WorkerID,
		name:       options.WorkerName,
		options:    options,
		etcdClient: etcdClient,
		taskChan:   make(chan *Task, options.BufferSize),
		status:     WorkerStatusOffline,
		stopChan:   make(chan struct{}),
		metrics:    &WorkerMetrics{},
	}

	return worker, nil
}

// SetTaskProcessor 设置任务处理器
func (w *WorkerNode) SetTaskProcessor(processor TaskProcessor) {
	w.processor = processor
}

// Start 启动工作节点
func (w *WorkerNode) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isStarted {
		return fmt.Errorf("worker already started")
	}

	if w.processor == nil {
		return fmt.Errorf("task processor not set")
	}

	log.Info().
		Str("workerId", w.id).
		Str("workerName", w.name).
		Msg("starting event-driven worker")

	// 创建租约
	if err := w.createLease(ctx); err != nil {
		return fmt.Errorf("failed to create lease: %w", err)
	}

	// 注册工作节点
	if err := w.registerWorker(ctx); err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	// 启动任务监听器
	w.wg.Add(1)
	go w.runTaskWatcher(ctx)

	// 启动任务处理器
	for i := 0; i < w.options.MaxConcurrency; i++ {
		w.wg.Add(1)
		go w.runTaskProcessor(ctx, i)
	}

	// 启动心跳
	w.wg.Add(1)
	go w.runHeartbeat(ctx)

	// 启动指标收集
	if w.options.EnableMetrics {
		w.wg.Add(1)
		go w.runMetricsCollection(ctx)
	}

	w.status = WorkerStatusOnline
	w.isStarted = true

	log.Info().
		Str("workerId", w.id).
		Msg("event-driven worker started successfully")

	return nil
}

// Stop 停止工作节点
func (w *WorkerNode) Stop(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isStarted {
		return fmt.Errorf("worker not started")
	}

	log.Info().Str("workerId", w.id).Msg("stopping event-driven worker")

	// 注销工作节点
	if err := w.unregisterWorker(ctx); err != nil {
		log.Error().Err(err).Msg("failed to unregister worker")
	}

	// 停止接收新任务
	close(w.stopChan)

	// 等待所有协程完成
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("all worker goroutines stopped gracefully")
	case <-time.After(30 * time.Second):
		log.Warn().Msg("timeout waiting for worker goroutines to stop")
	}

	// 撤销租约
	if w.leaseID != 0 {
		if _, err := w.etcdClient.Client.Revoke(ctx, w.leaseID); err != nil {
			log.Error().Err(err).Msg("failed to revoke worker lease")
		}
	}

	w.status = WorkerStatusOffline
	w.isStarted = false

	log.Info().Str("workerId", w.id).Msg("event-driven worker stopped")
	return nil
}

// GetID 获取工作节点ID
func (w *WorkerNode) GetID() string {
	return w.id
}

// GetStatus 获取工作节点状态
func (w *WorkerNode) GetStatus() WorkerStatus {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.status
}

// GetMetrics 获取工作节点指标
func (w *WorkerNode) GetMetrics() *WorkerMetrics {
	w.metrics.mu.RLock()
	defer w.metrics.mu.RUnlock()

	return &WorkerMetrics{
		TasksProcessed: w.metrics.TasksProcessed,
		TasksFailed:    w.metrics.TasksFailed,
		AvgDuration:    w.metrics.AvgDuration,
		LastTaskTime:   w.metrics.LastTaskTime,
	}
}

// createLease 创建ETCD租约
func (w *WorkerNode) createLease(ctx context.Context) error {
	ttl := int64(w.options.WorkerTTL.Seconds())
	resp, err := w.etcdClient.Client.Grant(ctx, ttl)
	if err != nil {
		return err
	}

	w.leaseID = resp.ID

	// 保持租约活跃
	keepAliveCh, err := w.etcdClient.Client.KeepAlive(ctx, w.leaseID)
	if err != nil {
		return err
	}

	// 消费KeepAlive响应
	go func() {
		for range keepAliveCh {
			// 保持租约活跃，无需处理响应
		}
	}()

	return nil
}

// registerWorker 注册工作节点
func (w *WorkerNode) registerWorker(ctx context.Context) error {
	worker := &Worker{
		ID:             w.id,
		Name:           w.name,
		Status:         WorkerStatusOnline,
		MaxConcurrency: w.options.MaxConcurrency,
		RegisteredAt:   currentTimestamp(),
		UpdatedAt:      currentTimestamp(),
		LastHeartbeat:  currentTimestamp(),
	}

	key := fmt.Sprintf("%sworkers/%s", w.options.EtcdKeyPrefix, w.id)
	data, err := json.Marshal(worker)
	if err != nil {
		return err
	}

	_, err = w.etcdClient.Client.Put(ctx, key, string(data), clientv3.WithLease(w.leaseID))
	return err
}

// unregisterWorker 注销工作节点
func (w *WorkerNode) unregisterWorker(ctx context.Context) error {
	key := fmt.Sprintf("%sworkers/%s", w.options.EtcdKeyPrefix, w.id)
	_, err := w.etcdClient.Client.Delete(ctx, key)
	return err
}

// runTaskWatcher 运行任务监听器
func (w *WorkerNode) runTaskWatcher(ctx context.Context) {
	defer w.wg.Done()

	// 监听分配给该工作节点的任务
	watchKey := fmt.Sprintf("%stasks/assigned/%s/", w.options.EtcdKeyPrefix, w.id)
	watchCh := w.etcdClient.Client.Watch(ctx, watchKey, clientv3.WithPrefix())

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopChan:
			return
		case watchResp := <-watchCh:
			w.handleTaskWatchEvents(ctx, watchResp)
		}
	}
}

// handleTaskWatchEvents 处理任务监听事件
func (w *WorkerNode) handleTaskWatchEvents(ctx context.Context, watchResp clientv3.WatchResponse) {
	for _, event := range watchResp.Events {
		switch event.Type {
		case clientv3.EventTypePut:
			var task Task
			if err := json.Unmarshal(event.Kv.Value, &task); err != nil {
				log.Error().Err(err).Msg("failed to unmarshal task")
				continue
			}

			// 将任务放入处理队列
			select {
			case w.taskChan <- &task:
				log.Info().
					Str("taskId", task.ID).
					Str("workerId", w.id).
					Msg("task received for processing")
			case <-ctx.Done():
				return
			case <-w.stopChan:
				return
			default:
				log.Warn().
					Str("taskId", task.ID).
					Msg("task queue full, dropping task")
			}

		case clientv3.EventTypeDelete:
			log.Debug().Str("key", string(event.Kv.Key)).Msg("task assignment deleted")
		}
	}
}

// runTaskProcessor 运行任务处理器
func (w *WorkerNode) runTaskProcessor(ctx context.Context, processorID int) {
	defer w.wg.Done()

	log.Debug().
		Str("workerId", w.id).
		Int("processorId", processorID).
		Msg("task processor started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopChan:
			return
		case task := <-w.taskChan:
			w.processTask(ctx, task)
		}
	}
}

// processTask 处理单个任务
func (w *WorkerNode) processTask(ctx context.Context, task *Task) {
	startTime := time.Now()

	log.Info().
		Str("taskId", task.ID).
		Str("workerId", w.id).
		Msg("starting task processing")

	// 更新任务状态为运行中
	task.Status = TaskStatusRunning
	task.StartedAt = currentTimestamp()
	task.UpdatedAt = currentTimestamp()

	if err := w.updateTaskStatus(ctx, task); err != nil {
		log.Error().Err(err).Str("taskId", task.ID).Msg("failed to update task status to running")
	}

	// 处理任务
	err := w.processor.Process(ctx, task)
	duration := time.Since(startTime)

	// 更新指标
	w.updateMetrics(err == nil, duration)

	// 更新任务状态
	if err != nil {
		task.Status = TaskStatusFailed
		task.Error = err.Error()
		log.Error().
			Err(err).
			Str("taskId", task.ID).
			Str("workerId", w.id).
			Dur("duration", duration).
			Msg("task processing failed")
	} else {
		task.Status = TaskStatusCompleted
		task.CompletedAt = currentTimestamp()
		log.Info().
			Str("taskId", task.ID).
			Str("workerId", w.id).
			Dur("duration", duration).
			Msg("task processing completed")
	}

	task.UpdatedAt = currentTimestamp()

	// 更新任务状态到ETCD
	if err := w.updateTaskStatus(ctx, task); err != nil {
		log.Error().Err(err).Str("taskId", task.ID).Msg("failed to update task final status")
	}

	// 删除任务分配记录
	assignedKey := fmt.Sprintf("%stasks/assigned/%s/%s", w.options.EtcdKeyPrefix, w.id, task.ID)
	if _, err := w.etcdClient.Client.Delete(ctx, assignedKey); err != nil {
		log.Error().Err(err).Str("taskId", task.ID).Msg("failed to delete task assignment")
	}
}

// updateTaskStatus 更新任务状态到ETCD
func (w *WorkerNode) updateTaskStatus(ctx context.Context, task *Task) error {
	key := fmt.Sprintf("%stasks/%s", w.options.EtcdKeyPrefix, task.ID)
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}

	_, err = w.etcdClient.Client.Put(ctx, key, string(data))
	return err
}

// runHeartbeat 运行心跳
func (w *WorkerNode) runHeartbeat(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.options.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.sendHeartbeat(ctx)
		}
	}
}

// sendHeartbeat 发送心跳
func (w *WorkerNode) sendHeartbeat(ctx context.Context) {
	worker := &Worker{
		ID:             w.id,
		Name:           w.name,
		Status:         w.status,
		MaxConcurrency: w.options.MaxConcurrency,
		LastHeartbeat:  currentTimestamp(),
		UpdatedAt:      currentTimestamp(),
	}

	key := fmt.Sprintf("%sworkers/%s", w.options.EtcdKeyPrefix, w.id)
	data, err := json.Marshal(worker)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal worker for heartbeat")
		return
	}

	_, err = w.etcdClient.Client.Put(ctx, key, string(data), clientv3.WithLease(w.leaseID))
	if err != nil {
		log.Error().Err(err).Msg("failed to send heartbeat")
	}
}

// runMetricsCollection 运行指标收集
func (w *WorkerNode) runMetricsCollection(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(60 * time.Second) // 每60秒收集一次
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopChan:
			return
		case <-ticker.C:
			// 可以在这里收集系统资源指标等
			log.Debug().
				Str("workerId", w.id).
				Int64("tasksProcessed", w.metrics.TasksProcessed).
				Int64("tasksFailed", w.metrics.TasksFailed).
				Msg("worker metrics collected")
		}
	}
}

// updateMetrics 更新指标
func (w *WorkerNode) updateMetrics(success bool, duration time.Duration) {
	w.metrics.mu.Lock()
	defer w.metrics.mu.Unlock()

	w.metrics.TasksProcessed++
	if !success {
		w.metrics.TasksFailed++
	}

	// 计算平均执行时间
	if w.metrics.TasksProcessed == 1 {
		w.metrics.AvgDuration = duration
	} else {
		w.metrics.AvgDuration = (w.metrics.AvgDuration + duration) / 2
	}

	w.metrics.LastTaskTime = currentTimestamp()
}
