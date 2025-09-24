package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kochabonline/kit/log"
	"github.com/redis/go-redis/v9"
)

// WorkerManager Redis工作节点管理器
type WorkerManager struct {
	client redis.Cmdable
	keys   *RedisKeys

	// 心跳管理
	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration

	// 停止通道
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewWorkerManager 创建工作节点管理器
func NewWorkerManager(client redis.Cmdable, keys *RedisKeys, heartbeatInterval, heartbeatTimeout time.Duration) *WorkerManager {
	return &WorkerManager{
		client:            client,
		keys:              keys,
		heartbeatInterval: heartbeatInterval,
		heartbeatTimeout:  heartbeatTimeout,
		stopChan:          make(chan struct{}),
	}
}

// RegisterWorker 注册工作节点
func (wm *WorkerManager) RegisterWorker(ctx context.Context, worker *Worker) error {
	worker.CreatedAt = time.Now().Unix()
	worker.UpdatedAt = worker.CreatedAt
	worker.LastHeartbeat = worker.CreatedAt
	worker.Status = WorkerStatusOnline
	worker.Version = 1

	// 序列化工作节点
	workerJSON, err := json.Marshal(worker)
	if err != nil {
		return fmt.Errorf("failed to marshal worker: %w", err)
	}

	pipe := wm.client.Pipeline()

	// 保存工作节点到Hash表
	pipe.HSet(ctx, wm.keys.WorkerHash, worker.ID, workerJSON)

	// 添加到在线工作节点集合
	pipe.SAdd(ctx, wm.keys.WorkerOnline, worker.ID)

	// 设置心跳键，过期时间为心跳超时时间
	heartbeatKey := wm.keys.WorkerHeartbeat + worker.ID
	pipe.SetEx(ctx, heartbeatKey, worker.ID, wm.heartbeatTimeout)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	log.Info().Str("workerId", worker.ID).Str("address", worker.Address).Msg("worker registered")
	return nil
}

// UnregisterWorker 注销工作节点
func (wm *WorkerManager) UnregisterWorker(ctx context.Context, workerID string) error {
	pipe := wm.client.Pipeline()

	// 从所有集合和哈希表中移除工作节点
	pipe.HDel(ctx, wm.keys.WorkerHash, workerID)
	pipe.SRem(ctx, wm.keys.WorkerOnline, workerID)
	pipe.SRem(ctx, wm.keys.WorkerBusy, workerID)

	// 删除心跳键
	heartbeatKey := wm.keys.WorkerHeartbeat + workerID
	pipe.Del(ctx, heartbeatKey)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to unregister worker: %w", err)
	}

	log.Info().Str("workerId", workerID).Msg("worker unregistered")
	return nil
}

// GetWorker 获取工作节点信息
func (wm *WorkerManager) GetWorker(ctx context.Context, workerID string) (*Worker, error) {
	result, err := wm.client.HGet(ctx, wm.keys.WorkerHash, workerID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrWorkerNotFound
		}
		return nil, fmt.Errorf("failed to get worker: %w", err)
	}

	var worker Worker
	if err := json.Unmarshal([]byte(result), &worker); err != nil {
		return nil, fmt.Errorf("failed to unmarshal worker: %w", err)
	}

	// 更新工作节点状态（检查心跳）
	if err := wm.updateWorkerStatus(ctx, &worker); err != nil {
		log.Error().Err(err).Str("workerId", workerID).Msg("failed to update worker status")
	}

	return &worker, nil
}

// ListWorkers 列出工作节点
func (wm *WorkerManager) ListWorkers(ctx context.Context, filter *WorkerFilter) ([]*Worker, error) {
	var workerIDs []string
	var err error

	if filter == nil {
		filter = &WorkerFilter{}
	}

	// 根据过滤条件获取工作节点ID列表
	if len(filter.Status) > 0 {
		workerIDs, err = wm.getWorkerIDsByStatus(ctx, filter.Status, filter.Limit, filter.Offset)
	} else if filter.OnlineOnly {
		workerIDs, err = wm.client.SMembers(ctx, wm.keys.WorkerOnline).Result()
	} else {
		workerIDs, err = wm.client.HKeys(ctx, wm.keys.WorkerHash).Result()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get worker IDs: %w", err)
	}

	// 应用分页
	if filter.Offset >= len(workerIDs) {
		return []*Worker{}, nil
	}

	end := filter.Offset + filter.Limit
	if filter.Limit > 0 && end < len(workerIDs) {
		workerIDs = workerIDs[filter.Offset:end]
	} else if filter.Offset > 0 {
		workerIDs = workerIDs[filter.Offset:]
	}

	// 批量获取工作节点详情
	workers := make([]*Worker, 0, len(workerIDs))
	for _, workerID := range workerIDs {
		worker, err := wm.GetWorker(ctx, workerID)
		if err != nil {
			continue // 忽略获取失败的工作节点
		}

		// 应用其他过滤条件
		if wm.matchFilter(worker, filter) {
			workers = append(workers, worker)
		}
	}

	return workers, nil
}

// UpdateWorkerStatus 更新工作节点状态
func (wm *WorkerManager) UpdateWorkerStatus(ctx context.Context, workerID string, status WorkerStatus) error {
	worker, err := wm.GetWorker(ctx, workerID)
	if err != nil {
		return err
	}

	oldStatus := worker.Status
	worker.Status = status
	worker.UpdatedAt = time.Now().Unix()
	worker.Version++

	// 序列化工作节点
	workerJSON, err := json.Marshal(worker)
	if err != nil {
		return fmt.Errorf("failed to marshal worker: %w", err)
	}

	pipe := wm.client.Pipeline()

	// 更新工作节点数据
	pipe.HSet(ctx, wm.keys.WorkerHash, workerID, workerJSON)

	// 更新状态集合
	switch oldStatus {
	case WorkerStatusOnline:
		pipe.SRem(ctx, wm.keys.WorkerOnline, workerID)
	case WorkerStatusBusy:
		pipe.SRem(ctx, wm.keys.WorkerBusy, workerID)
	}

	switch status {
	case WorkerStatusOnline:
		pipe.SAdd(ctx, wm.keys.WorkerOnline, workerID)
		pipe.SRem(ctx, wm.keys.WorkerBusy, workerID)
	case WorkerStatusBusy:
		pipe.SAdd(ctx, wm.keys.WorkerBusy, workerID)
		pipe.SRem(ctx, wm.keys.WorkerOnline, workerID)
	case WorkerStatusOffline:
		pipe.SRem(ctx, wm.keys.WorkerOnline, workerID)
		pipe.SRem(ctx, wm.keys.WorkerBusy, workerID)
	}

	_, err = pipe.Exec(ctx)
	return err
}

// SendHeartbeat 发送心跳
func (wm *WorkerManager) SendHeartbeat(ctx context.Context, workerID string, currentTasks int, cpu, memory, load float64) error {
	// 更新工作节点信息
	worker, err := wm.GetWorker(ctx, workerID)
	if err != nil {
		return err
	}

	worker.LastHeartbeat = time.Now().Unix()
	worker.CurrentTasks = currentTasks
	worker.CPU = cpu
	worker.Memory = memory
	worker.Load = load
	worker.UpdatedAt = worker.LastHeartbeat

	// 根据当前任务数更新状态
	if currentTasks >= worker.MaxTasks {
		worker.Status = WorkerStatusBusy
	} else {
		worker.Status = WorkerStatusOnline
	}

	// 序列化工作节点
	workerJSON, err := json.Marshal(worker)
	if err != nil {
		return fmt.Errorf("failed to marshal worker: %w", err)
	}

	pipe := wm.client.Pipeline()

	// 更新工作节点数据
	pipe.HSet(ctx, wm.keys.WorkerHash, workerID, workerJSON)

	// 更新心跳键
	heartbeatKey := wm.keys.WorkerHeartbeat + workerID
	pipe.SetEx(ctx, heartbeatKey, workerID, wm.heartbeatTimeout)

	// 更新状态集合
	if worker.Status == WorkerStatusBusy {
		pipe.SAdd(ctx, wm.keys.WorkerBusy, workerID)
		pipe.SRem(ctx, wm.keys.WorkerOnline, workerID)
	} else {
		pipe.SAdd(ctx, wm.keys.WorkerOnline, workerID)
		pipe.SRem(ctx, wm.keys.WorkerBusy, workerID)
	}

	_, err = pipe.Exec(ctx)
	return err
}

// GetOnlineWorkers 获取在线工作节点
func (wm *WorkerManager) GetOnlineWorkers(ctx context.Context) ([]*Worker, error) {
	workerIDs, err := wm.client.SMembers(ctx, wm.keys.WorkerOnline).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get online workers: %w", err)
	}

	workers := make([]*Worker, 0, len(workerIDs))
	for _, workerID := range workerIDs {
		worker, err := wm.GetWorker(ctx, workerID)
		if err != nil {
			continue
		}

		// 双重检查工作节点是否真的在线（检查心跳）
		if worker.Status == WorkerStatusOnline || worker.Status == WorkerStatusBusy {
			workers = append(workers, worker)
		}
	}

	return workers, nil
}

// GetAvailableWorkers 获取可用工作节点（在线且未满负荷）
func (wm *WorkerManager) GetAvailableWorkers(ctx context.Context) ([]*Worker, error) {
	onlineWorkers, err := wm.GetOnlineWorkers(ctx)
	if err != nil {
		return nil, err
	}

	var availableWorkers []*Worker
	for _, worker := range onlineWorkers {
		if worker.CurrentTasks < worker.MaxTasks {
			availableWorkers = append(availableWorkers, worker)
		}
	}

	return availableWorkers, nil
}

// StartHeartbeatMonitor 启动心跳监控
func (wm *WorkerManager) StartHeartbeatMonitor(ctx context.Context) {
	wm.wg.Add(1)
	go wm.runHeartbeatMonitor(ctx)
}

// Stop 停止工作节点管理器
func (wm *WorkerManager) Stop() {
	close(wm.stopChan)
	wm.wg.Wait()
}

// runHeartbeatMonitor 运行心跳监控
func (wm *WorkerManager) runHeartbeatMonitor(ctx context.Context) {
	defer wm.wg.Done()

	ticker := time.NewTicker(wm.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-wm.stopChan:
			return
		case <-ticker.C:
			if err := wm.checkWorkerHeartbeats(ctx); err != nil {
				log.Error().Err(err).Msg("failed to check worker heartbeats")
			}
		}
	}
}

// checkWorkerHeartbeats 检查工作节点心跳
func (wm *WorkerManager) checkWorkerHeartbeats(ctx context.Context) error {
	// 获取所有注册的工作节点
	workerIDs, err := wm.client.HKeys(ctx, wm.keys.WorkerHash).Result()
	if err != nil {
		return fmt.Errorf("failed to get worker IDs: %w", err)
	}

	for _, workerID := range workerIDs {
		// 检查心跳键是否存在
		heartbeatKey := wm.keys.WorkerHeartbeat + workerID
		exists, err := wm.client.Exists(ctx, heartbeatKey).Result()
		if err != nil {
			log.Error().Err(err).Str("workerId", workerID).Msg("failed to check heartbeat")
			continue
		}

		if exists == 0 {
			// 心跳超时，标记工作节点为离线
			if err := wm.UpdateWorkerStatus(ctx, workerID, WorkerStatusOffline); err != nil {
				log.Error().Err(err).Str("workerId", workerID).Msg("failed to mark worker as offline")
			} else {
				log.Warn().Str("workerId", workerID).Msg("worker marked as offline due to heartbeat timeout")
			}
		}
	}

	return nil
}

// updateWorkerStatus 更新工作节点状态（基于心跳检查）
func (wm *WorkerManager) updateWorkerStatus(ctx context.Context, worker *Worker) error {
	// 检查心跳
	heartbeatKey := wm.keys.WorkerHeartbeat + worker.ID
	exists, err := wm.client.Exists(ctx, heartbeatKey).Result()
	if err != nil {
		return err
	}

	if exists == 0 && (worker.Status == WorkerStatusOnline || worker.Status == WorkerStatusBusy) {
		// 心跳超时，但工作节点状态仍为在线，需要更新
		worker.Status = WorkerStatusOffline
		worker.UpdatedAt = time.Now().Unix()

		// 保存更新后的状态
		workerJSON, err := json.Marshal(worker)
		if err != nil {
			return err
		}

		pipe := wm.client.Pipeline()
		pipe.HSet(ctx, wm.keys.WorkerHash, worker.ID, workerJSON)
		pipe.SRem(ctx, wm.keys.WorkerOnline, worker.ID)
		pipe.SRem(ctx, wm.keys.WorkerBusy, worker.ID)

		_, err = pipe.Exec(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// getWorkerIDsByStatus 根据状态获取工作节点ID列表
func (wm *WorkerManager) getWorkerIDsByStatus(ctx context.Context, statuses []WorkerStatus, limit, offset int) ([]string, error) {
	var allIDs []string

	for _, status := range statuses {
		var setKey string

		switch status {
		case WorkerStatusOnline:
			setKey = wm.keys.WorkerOnline
		case WorkerStatusBusy:
			setKey = wm.keys.WorkerBusy
		case WorkerStatusOffline:
			// 对于离线状态，需要从所有工作节点中筛选
			allWorkerIDs, err := wm.client.HKeys(ctx, wm.keys.WorkerHash).Result()
			if err != nil {
				continue
			}

			onlineIDs, _ := wm.client.SMembers(ctx, wm.keys.WorkerOnline).Result()
			busyIDs, _ := wm.client.SMembers(ctx, wm.keys.WorkerBusy).Result()

			onlineSet := make(map[string]bool)
			for _, id := range onlineIDs {
				onlineSet[id] = true
			}
			for _, id := range busyIDs {
				onlineSet[id] = true
			}

			for _, id := range allWorkerIDs {
				if !onlineSet[id] {
					allIDs = append(allIDs, id)
				}
			}
			continue
		}

		if setKey != "" {
			ids, err := wm.client.SMembers(ctx, setKey).Result()
			if err != nil {
				continue
			}
			allIDs = append(allIDs, ids...)
		}
	}

	// 去重并应用分页
	uniqueIDs := wm.removeDuplicates(allIDs)

	if offset >= len(uniqueIDs) {
		return []string{}, nil
	}

	end := offset + limit
	if limit > 0 && end < len(uniqueIDs) {
		end = len(uniqueIDs)
	} else if limit <= 0 {
		end = len(uniqueIDs)
	}

	return uniqueIDs[offset:end], nil
}

// matchFilter 检查工作节点是否匹配过滤条件
func (wm *WorkerManager) matchFilter(worker *Worker, filter *WorkerFilter) bool {
	// 能力过滤
	if len(filter.Capabilities) > 0 {
		workerCapabilities := make(map[string]bool)
		for _, cap := range worker.Capabilities {
			workerCapabilities[cap] = true
		}

		for _, requiredCap := range filter.Capabilities {
			if !workerCapabilities[requiredCap] {
				return false
			}
		}
	}

	// 标签过滤
	if len(filter.Tags) > 0 {
		workerTags := make(map[string]bool)
		for _, tag := range worker.Tags {
			workerTags[tag] = true
		}

		for _, requiredTag := range filter.Tags {
			if !workerTags[requiredTag] {
				return false
			}
		}
	}

	// 负载过滤
	if filter.MinLoad > 0 && worker.Load < filter.MinLoad {
		return false
	}

	if filter.MaxLoad > 0 && worker.Load > filter.MaxLoad {
		return false
	}

	return true
}

// removeDuplicates 去除重复的工作节点ID
func (wm *WorkerManager) removeDuplicates(ids []string) []string {
	seen := make(map[string]bool)
	var unique []string

	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}

	return unique
}

// GetWorkerCount 获取工作节点数量统计
func (wm *WorkerManager) GetWorkerCount(ctx context.Context) (map[WorkerStatus]int64, error) {
	pipe := wm.client.Pipeline()

	totalCmd := pipe.HLen(ctx, wm.keys.WorkerHash)
	onlineCmd := pipe.SCard(ctx, wm.keys.WorkerOnline)
	busyCmd := pipe.SCard(ctx, wm.keys.WorkerBusy)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	total := totalCmd.Val()
	online := onlineCmd.Val()
	busy := busyCmd.Val()
	offline := total - online - busy
	idle := online - busy // 在线但不忙碌的节点

	return map[WorkerStatus]int64{
		WorkerStatusOnline:  idle,
		WorkerStatusBusy:    busy,
		WorkerStatusOffline: offline,
	}, nil
} // UpdateWorkerTaskCount 更新工作节点任务数量
func (wm *WorkerManager) UpdateWorkerTaskCount(ctx context.Context, workerID string, currentTasks int) error {
	worker, err := wm.GetWorker(ctx, workerID)
	if err != nil {
		return err
	}

	worker.CurrentTasks = currentTasks
	worker.UpdatedAt = time.Now().Unix()

	// 根据任务数量更新状态
	if currentTasks >= worker.MaxTasks {
		worker.Status = WorkerStatusBusy
	} else {
		worker.Status = WorkerStatusOnline
	}

	return wm.saveWorker(ctx, worker)
}

// saveWorker 保存工作节点到Redis
func (wm *WorkerManager) saveWorker(ctx context.Context, worker *Worker) error {
	workerJSON, err := json.Marshal(worker)
	if err != nil {
		return fmt.Errorf("failed to marshal worker: %w", err)
	}

	pipe := wm.client.Pipeline()

	// 更新工作节点数据
	pipe.HSet(ctx, wm.keys.WorkerHash, worker.ID, workerJSON)

	// 更新状态集合
	switch worker.Status {
	case WorkerStatusOnline:
		pipe.SAdd(ctx, wm.keys.WorkerOnline, worker.ID)
		pipe.SRem(ctx, wm.keys.WorkerBusy, worker.ID)
	case WorkerStatusBusy:
		pipe.SAdd(ctx, wm.keys.WorkerBusy, worker.ID)
		pipe.SRem(ctx, wm.keys.WorkerOnline, worker.ID)
	case WorkerStatusOffline:
		pipe.SRem(ctx, wm.keys.WorkerOnline, worker.ID)
		pipe.SRem(ctx, wm.keys.WorkerBusy, worker.ID)
	}

	_, err = pipe.Exec(ctx)
	return err
}
