package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kochabonline/kit/log"
	"github.com/redis/go-redis/v9"
)

// WorkerManager Redis工作节点管理器
type WorkerManager struct {
	client redis.Cmdable
	keys   *RedisKeys

	leaseTTL    time.Duration
	leaseCancel map[string]context.CancelFunc
	leaseMutex  sync.RWMutex
}

// NewWorkerManager 创建工作节点管理器
func NewWorkerManager(client redis.Cmdable, keys *RedisKeys, leaseTTL time.Duration) *WorkerManager {
	// 强制最小 TTL，避免 go-redis 或外部使用中出现过小值导致频繁续租或被截断日志
	const minTTL = time.Second
	if leaseTTL < minTTL {
		leaseTTL = minTTL
	}
	return &WorkerManager{
		client:      client,
		keys:        keys,
		leaseTTL:    leaseTTL,
		leaseCancel: make(map[string]context.CancelFunc),
	}
}

// ================================================================================
// 工作节点生命周期管理
// ================================================================================

// RegisterWorker 注册工作节点
func (wm *WorkerManager) RegisterWorker(ctx context.Context, worker *Worker) error {
	if worker == nil || worker.ID == "" {
		return errors.New("invalid worker: empty")
	}
	now := time.Now().Unix()
	worker.CreatedAt = now
	worker.UpdatedAt = now
	if worker.Status == 0 { // 默认设置为在线
		worker.Status = WorkerStatusOnline
	}
	if worker.MaxTasks == 0 {
		worker.MaxTasks = 1
	}
	worker.Version = 1

	if err := wm.persistWorker(ctx, worker); err != nil {
		return fmt.Errorf("register worker persist failed: %w", err)
	}

	cancel, err := wm.KeepAlive(ctx, worker.ID)
	if err != nil {
		return fmt.Errorf("start auto renew failed: %w", err)
	}
	wm.storeLeaseCancel(worker.ID, cancel)
	log.Info().Str("workerId", worker.ID).Str("address", worker.Address).Msg("worker registered")
	return nil
}

// UnregisterWorker 注销工作节点
func (wm *WorkerManager) UnregisterWorker(ctx context.Context, workerID string) error {
	return wm.gracefulShutdown(ctx, workerID, "manual unregistration")
}

// GracefulShutdown 工作节点优雅关闭 - 立即清理而不等待租约过期
func (wm *WorkerManager) GracefulShutdown(ctx context.Context, workerID string) error {
	return wm.gracefulShutdown(ctx, workerID, "graceful shutdown")
}

// gracefulShutdown 内部优雅关闭逻辑
func (wm *WorkerManager) gracefulShutdown(ctx context.Context, workerID string, reason string) error {
	wm.stopLeaseRenewal(workerID)
	worker, err := wm.GetWorker(ctx, workerID)
	if err != nil {
		if err == ErrWorkerNotFound {
			log.Warn().Str("workerId", workerID).Str("reason", reason).Msg("worker not found during shutdown")
			return nil
		}
		return fmt.Errorf("fetch worker for shutdown failed: %w", err)
	}
	worker.Status = WorkerStatusOffline
	worker.UpdatedAt = time.Now().Unix()

	// 直接删除键，避免冗余写放大；如果需要审计，可扩展事件。
	pipe := wm.client.Pipeline()
	pipe.Del(ctx, wm.workerKey(workerID))
	pipe.Del(ctx, wm.onlineKey(workerID))
	pipe.Del(ctx, wm.busyKey(workerID))
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis cleanup failed: %w", err)
	}
	log.Info().Str("workerId", workerID).Str("reason", reason).Str("address", worker.Address).Msg("worker shutdown")
	return nil
}

// GetWorker 获取工作节点信息
func (wm *WorkerManager) GetWorker(ctx context.Context, workerID string) (*Worker, error) {
	if workerID == "" {
		return nil, ErrWorkerNotFound
	}
	result, err := wm.client.Get(ctx, wm.workerKey(workerID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrWorkerNotFound
		}
		return nil, fmt.Errorf("get worker failed: %w", err)
	}
	var w Worker
	if err := json.Unmarshal([]byte(result), &w); err != nil {
		return nil, fmt.Errorf("decode worker failed: %w", err)
	}
	return &w, nil
}

// ListWorkers 列出工作节点
func (wm *WorkerManager) ListWorkers(ctx context.Context, filter *WorkerFilter) ([]*Worker, error) {
	if filter == nil {
		filter = &WorkerFilter{}
	}

	// 1. 获取候选ID
	ids, err := wm.resolveIDs(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []*Worker{}, nil
	}

	// 2. 批量加载
	workers, _ := wm.batchGet(ctx, ids)

	// 3. 过滤
	var result []*Worker
	for _, w := range workers {
		if w == nil {
			continue
		}
		if !wm.applyFilter(w, filter) {
			continue
		}
		result = append(result, w)
	}

	// 4. 分页（在过滤后）
	if filter.Offset >= len(result) {
		return []*Worker{}, nil
	}
	end := len(result)
	if filter.Limit > 0 && filter.Offset+filter.Limit < end {
		end = filter.Offset + filter.Limit
	}
	if filter.Offset > 0 || end < len(result) {
		result = result[filter.Offset:end]
	}
	return result, nil
}

// ================================================================================
// 工作节点状态和心跳管理
// ================================================================================

// UpdateWorkerStatus 更新工作节点状态
func (wm *WorkerManager) UpdateWorkerStatus(ctx context.Context, workerID string, status WorkerStatus) error {
	w, err := wm.GetWorker(ctx, workerID)
	if err != nil {
		return err
	}
	w.Status = status
	w.UpdatedAt = time.Now().Unix()
	w.Version++
	return wm.persistWorker(ctx, w)
}

// ================================================================================
// 工作节点查询和统计方法
// ================================================================================

// getWorkerIDsByPattern 根据模式扫描工作节点键并提取ID
func (wm *WorkerManager) getWorkerIDsByPattern(ctx context.Context, pattern string) ([]string, error) {
	var (
		cursor uint64
		ids    []string
	)
	for {
		keys, next, err := wm.client.Scan(ctx, cursor, pattern, 256).Result()
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			if idx := strings.LastIndexByte(k, ':'); idx > -1 {
				ids = append(ids, k[idx+1:])
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return ids, nil
}

// getOnlineWorkerIDs 获取在线工作节点ID列表
func (wm *WorkerManager) getOnlineWorkerIDs(ctx context.Context) ([]string, error) {
	pattern := wm.keys.WorkerOnline + ":*"
	return wm.getWorkerIDsByPattern(ctx, pattern)
}

// getBusyWorkerIDs 获取忙碌工作节点ID列表
func (wm *WorkerManager) getBusyWorkerIDs(ctx context.Context) ([]string, error) {
	pattern := wm.keys.WorkerBusy + ":*"
	return wm.getWorkerIDsByPattern(ctx, pattern)
}

// getAllWorkerIDs 获取所有工作节点ID列表
func (wm *WorkerManager) getAllWorkerIDs(ctx context.Context) ([]string, error) {
	pattern := wm.keys.WorkerHash + ":*"
	return wm.getWorkerIDsByPattern(ctx, pattern)
}

// GetOnlineWorkers 获取在线工作节点
func (wm *WorkerManager) GetOnlineWorkers(ctx context.Context) ([]*Worker, error) {
	ids, err := wm.getOnlineWorkerIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("online ids fetch failed: %w", err)
	}
	ws, _ := wm.batchGet(ctx, ids)
	var ret []*Worker
	for _, w := range ws {
		if w == nil {
			continue
		}
		if w.Status == WorkerStatusOnline || w.Status == WorkerStatusBusy || w.Status == WorkerStatusIdle {
			ret = append(ret, w)
		}
	}
	return ret, nil
}

// GetAvailableWorkers 获取可用工作节点（在线且未满负荷）
func (wm *WorkerManager) GetAvailableWorkers(ctx context.Context) ([]*Worker, error) {
	ws, err := wm.GetOnlineWorkers(ctx)
	if err != nil {
		return nil, err
	}
	var ret []*Worker
	for _, w := range ws {
		if w.CurrentTasks < w.MaxTasks {
			ret = append(ret, w)
		}
	}
	return ret, nil
}

// ================================================================================
// 监控和清理机制 (基于Redis TTL自动清理)
// ================================================================================

// Stop 停止工作节点管理器（只停止续租协程）
func (wm *WorkerManager) Stop() {
	wm.leaseMutex.Lock()
	for workerID, cancel := range wm.leaseCancel {
		cancel()
		log.Debug().Str("workerId", workerID).Msg("stopped lease renewal during shutdown")
	}
	wm.leaseCancel = make(map[string]context.CancelFunc)
	wm.leaseMutex.Unlock()
}

// GetWorkerCount 获取工作节点数量统计
func (wm *WorkerManager) GetWorkerCount(ctx context.Context) (map[WorkerStatus]int64, error) {
	// 使用新的方式获取各类型节点数量
	allWorkerIDs, err := wm.getAllWorkerIDs(ctx)
	if err != nil {
		return nil, err
	}

	onlineWorkerIDs, err := wm.getOnlineWorkerIDs(ctx)
	if err != nil {
		return nil, err
	}

	busyWorkerIDs, err := wm.getBusyWorkerIDs(ctx)
	if err != nil {
		return nil, err
	}

	total := int64(len(allWorkerIDs))
	online := int64(len(onlineWorkerIDs))
	busy := int64(len(busyWorkerIDs))
	offline := total - online - busy
	idle := online - busy // 在线但不忙碌的节点

	return map[WorkerStatus]int64{
		WorkerStatusOnline:  idle,
		WorkerStatusBusy:    busy,
		WorkerStatusOffline: offline,
	}, nil
}

// UpdateWorkerTaskCount 更新工作节点任务数量
func (wm *WorkerManager) UpdateWorkerTaskCount(ctx context.Context, workerID string, currentTasks int) error {
	w, err := wm.GetWorker(ctx, workerID)
	if err != nil {
		return err
	}
	w.CurrentTasks = currentTasks
	w.UpdatedAt = time.Now().Unix()
	switch {
	case currentTasks >= w.MaxTasks:
		w.Status = WorkerStatusBusy
	case currentTasks == 0:
		w.Status = WorkerStatusIdle
	default:
		w.Status = WorkerStatusOnline
	}
	return wm.persistWorker(ctx, w)
}

// ================================================================================
// 内部工具方法
// ================================================================================

// saveWorker 保存工作节点到Redis

// stopLeaseRenewal 停止工作节点的续租协程
func (wm *WorkerManager) stopLeaseRenewal(workerID string) {
	wm.leaseMutex.Lock()
	defer wm.leaseMutex.Unlock()

	if cancel, exists := wm.leaseCancel[workerID]; exists {
		cancel()
		delete(wm.leaseCancel, workerID)
		log.Debug().Str("workerId", workerID).Msg("stopped lease renewal for worker")
	}
}

// saveWorkerWithLease 保存工作节点到Redis（包含租约续期）
func (wm *WorkerManager) persistWorker(ctx context.Context, w *Worker) error {
	if w == nil {
		return errors.New("nil worker")
	}
	data, err := json.Marshal(w)
	if err != nil {
		return err
	}
	pipe := wm.client.Pipeline()
	pipe.SetEx(ctx, wm.workerKey(w.ID), data, wm.leaseTTL)
	// 清理旧状态键
	pipe.Del(ctx, wm.onlineKey(w.ID))
	pipe.Del(ctx, wm.busyKey(w.ID))
	switch w.Status {
	case WorkerStatusOnline, WorkerStatusIdle:
		pipe.SetEx(ctx, wm.onlineKey(w.ID), w.ID, wm.leaseTTL)
	case WorkerStatusBusy:
		pipe.SetEx(ctx, wm.busyKey(w.ID), w.ID, wm.leaseTTL)
	}
	_, err = pipe.Exec(ctx)
	return err
}

// performDeepCleanup 执行轻量级监控检查，大部分清理工作由Redis TTL自动完成
// 已删除后台周期清理；如需统计可直接调用 GetWorkerCount。

// ================================================================================
// 租约管理 (类似etcd的lease机制)
// ================================================================================

// RenewLease 工作节点续期租约，类似于etcd的KeepAlive
func (wm *WorkerManager) RenewLease(ctx context.Context, workerID string) error {
	w, err := wm.GetWorker(ctx, workerID)
	if err != nil {
		return fmt.Errorf("renew: %w", err)
	}
	w.UpdatedAt = time.Now().Unix()
	return wm.persistWorker(ctx, w)
}

// KeepAlive 启动工作节点的自动续约机制（自动续租）
// 返回一个取消函数，调用它可以停止续租
func (wm *WorkerManager) KeepAlive(ctx context.Context, workerID string) (context.CancelFunc, error) {
	if _, err := wm.GetWorker(ctx, workerID); err != nil {
		return nil, err
	}
	renewCtx, cancel := context.WithCancel(ctx)
	interval := wm.leaseTTL / 3
	if interval < time.Second {
		interval = time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		log.Debug().Str("workerId", workerID).Dur("interval", interval).Msg("auto renew started")
		for {
			select {
			case <-renewCtx.Done():
				return
			case <-ticker.C:
				if err := wm.RenewLease(renewCtx, workerID); err != nil && renewCtx.Err() == nil {
					log.Error().Err(err).Str("workerId", workerID).Msg("lease renew failed")
				}
			}
		}
	}()
	return cancel, nil
}

// GetLeaseInfo 获取工作节点租约信息
func (wm *WorkerManager) GetLeaseInfo(ctx context.Context, workerID string) (*WorkerLeaseInfo, error) {
	w, err := wm.GetWorker(ctx, workerID)
	if err != nil {
		return nil, err
	}
	ttl, err := wm.client.TTL(ctx, wm.workerKey(workerID)).Result()
	if err != nil {
		return nil, err
	}
	return &WorkerLeaseInfo{
		WorkerID:       workerID,
		IsActive:       ttl > 0,
		TTLRemaining:   ttl,
		LastRenewal:    time.Unix(w.UpdatedAt, 0),
		Status:         w.Status,
		LeaseExpiredAt: time.Now().Add(ttl),
		ConfiguredTTL:  wm.leaseTTL,
		RenewInterval:  wm.leaseTTL / 3,
	}, nil
}

// GetAllLeaseInfo 获取所有工作节点的租约信息
func (wm *WorkerManager) GetAllLeaseInfo(ctx context.Context) ([]*WorkerLeaseInfo, error) {
	ids, err := wm.getAllWorkerIDs(ctx)
	if err != nil {
		return nil, err
	}
	var infos []*WorkerLeaseInfo
	for _, id := range ids {
		li, err := wm.GetLeaseInfo(ctx, id)
		if err != nil {
			continue
		}
		infos = append(infos, li)
	}
	return infos, nil
}

// ============================= Helper / Internal ==============================
func (wm *WorkerManager) workerKey(id string) string { return wm.keys.WorkerHash + ":" + id }
func (wm *WorkerManager) onlineKey(id string) string { return wm.keys.WorkerOnline + ":" + id }
func (wm *WorkerManager) busyKey(id string) string   { return wm.keys.WorkerBusy + ":" + id }

func (wm *WorkerManager) storeLeaseCancel(id string, cancel context.CancelFunc) {
	wm.leaseMutex.Lock()
	defer wm.leaseMutex.Unlock()
	wm.leaseCancel[id] = cancel
}

// 批量获取 worker (best-effort) 返回与输入id顺序一致切片
func (wm *WorkerManager) batchGet(ctx context.Context, ids []string) ([]*Worker, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, wm.workerKey(id))
	}
	vals, err := wm.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	res := make([]*Worker, 0, len(vals))
	for i, v := range vals {
		if v == nil {
			res = append(res, nil)
			continue
		}
		s, ok := v.(string)
		if !ok {
			res = append(res, nil)
			continue
		}
		var w Worker
		if err := json.Unmarshal([]byte(s), &w); err != nil {
			res = append(res, nil)
			continue
		}
		// 防御: 确保ID一致
		if w.ID == "" {
			w.ID = ids[i]
		}
		res = append(res, &w)
	}
	return res, nil
}

// 根据过滤器解析待查询的ID集合（不做高级索引，仅使用现有 key 模式）
func (wm *WorkerManager) resolveIDs(ctx context.Context, f *WorkerFilter) ([]string, error) {
	if f == nil {
		return wm.getAllWorkerIDs(ctx)
	}
	if len(f.Status) == 0 && !f.OnlineOnly {
		return wm.getAllWorkerIDs(ctx)
	}
	if f.OnlineOnly {
		return wm.getOnlineWorkerIDs(ctx)
	}
	return wm.getWorkerIDsByStatus(ctx, f.Status, 0, 0)
}

// 过滤器匹配（替换旧 matchFilter）
func (wm *WorkerManager) applyFilter(w *Worker, f *WorkerFilter) bool {
	if f == nil {
		return true
	}
	// 状态: 如果指定 Status 集合则检查
	if len(f.Status) > 0 {
		ok := false
		for _, st := range f.Status {
			if w.Status == st {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if len(f.Capabilities) > 0 {
		capSet := make(map[string]struct{}, len(w.Capabilities))
		for _, c := range w.Capabilities {
			capSet[c] = struct{}{}
		}
		for _, need := range f.Capabilities {
			if _, ok := capSet[need]; !ok {
				return false
			}
		}
	}
	if len(f.Tags) > 0 {
		tagSet := make(map[string]struct{}, len(w.Tags))
		for _, t := range w.Tags {
			tagSet[t] = struct{}{}
		}
		for _, t := range f.Tags {
			if _, ok := tagSet[t]; !ok {
				return false
			}
		}
	}
	if f.MinLoad > 0 && w.Load < f.MinLoad {
		return false
	}
	if f.MaxLoad > 0 && w.Load > f.MaxLoad {
		return false
	}
	return true
}

// getWorkerIDsByStatus 保留逻辑（分页由上层处理）
// offline 通过差集判定
func (wm *WorkerManager) getWorkerIDsByStatus(ctx context.Context, statuses []WorkerStatus, limit, offset int) ([]string, error) { // 复用现有实现的一部分
	var all []string
	for _, st := range statuses {
		switch st {
		case WorkerStatusOnline, WorkerStatusIdle:
			ids, err := wm.getOnlineWorkerIDs(ctx)
			if err == nil {
				all = append(all, ids...)
			}
		case WorkerStatusBusy:
			ids, err := wm.getBusyWorkerIDs(ctx)
			if err == nil {
				all = append(all, ids...)
			}
		case WorkerStatusOffline:
			allIDs, err := wm.getAllWorkerIDs(ctx)
			if err != nil {
				continue
			}
			online, _ := wm.getOnlineWorkerIDs(ctx)
			busy, _ := wm.getBusyWorkerIDs(ctx)
			live := make(map[string]struct{})
			for _, id := range online {
				live[id] = struct{}{}
			}
			for _, id := range busy {
				live[id] = struct{}{}
			}
			for _, id := range allIDs {
				if _, ok := live[id]; !ok {
					all = append(all, id)
				}
			}
		}
	}
	// 去重
	uniq := make(map[string]struct{}, len(all))
	ordered := make([]string, 0, len(all))
	for _, id := range all {
		if _, ok := uniq[id]; !ok {
			uniq[id] = struct{}{}
			ordered = append(ordered, id)
		}
	}
	return ordered, nil
}
