package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// TaskManager Redis任务管理器
type TaskManager struct {
	client redis.Cmdable
	keys   *RedisKeys
}

// NewTaskManager 创建任务管理器
func NewTaskManager(client redis.Cmdable, keys *RedisKeys) *TaskManager {
	return &TaskManager{
		client: client,
		keys:   keys,
	}
}

// SaveTask 保存任务到Redis
func (tm *TaskManager) SaveTask(ctx context.Context, task *Task) error {
	// 序列化任务
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	pipe := tm.client.Pipeline()

	// 保存任务到Hash表
	pipe.HSet(ctx, tm.keys.TaskHash, task.ID, taskJSON)

	// 根据任务状态更新相应的集合
	switch task.Status {
	case TaskStatusPending:
		// 添加到优先级队列 (ZSet)，分数为优先级 + 时间戳的组合
		score := tm.calculatePriorityScore(task.Priority, task.CreatedAt)
		pipe.ZAdd(ctx, tm.keys.TaskPriorityQueue, redis.Z{
			Score:  score,
			Member: task.ID,
		})

		// 添加到普通队列 (List) - FIFO
		pipe.LPush(ctx, tm.keys.TaskQueue, task.ID)

	case TaskStatusScheduled:
		pipe.SAdd(ctx, tm.keys.TaskScheduled, task.ID)

	case TaskStatusRunning:
		pipe.SAdd(ctx, tm.keys.TaskRunning, task.ID)

	case TaskStatusCompleted:
		pipe.SAdd(ctx, tm.keys.TaskCompleted, task.ID)
		// 从运行集合中移除
		pipe.SRem(ctx, tm.keys.TaskRunning, task.ID)

	case TaskStatusFailed:
		pipe.SAdd(ctx, tm.keys.TaskFailed, task.ID)
		// 从运行集合中移除
		pipe.SRem(ctx, tm.keys.TaskRunning, task.ID)

	case TaskStatusCanceled:
		// 从所有队列和集合中移除
		pipe.ZRem(ctx, tm.keys.TaskPriorityQueue, task.ID)
		pipe.LRem(ctx, tm.keys.TaskQueue, 0, task.ID)
		pipe.SRem(ctx, tm.keys.TaskScheduled, task.ID)
		pipe.SRem(ctx, tm.keys.TaskRunning, task.ID)
	}

	_, err = pipe.Exec(ctx)
	return err
}

// GetTask 从Redis获取任务
func (tm *TaskManager) GetTask(ctx context.Context, taskID string) (*Task, error) {
	result, err := tm.client.HGet(ctx, tm.keys.TaskHash, taskID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	var task Task
	if err := json.Unmarshal([]byte(result), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

// DeleteTask 删除任务
func (tm *TaskManager) DeleteTask(ctx context.Context, taskID string) error {
	pipe := tm.client.Pipeline()

	// 从所有数据结构中移除任务
	pipe.HDel(ctx, tm.keys.TaskHash, taskID)
	pipe.ZRem(ctx, tm.keys.TaskPriorityQueue, taskID)
	pipe.LRem(ctx, tm.keys.TaskQueue, 0, taskID)
	pipe.SRem(ctx, tm.keys.TaskScheduled, taskID)
	pipe.SRem(ctx, tm.keys.TaskRunning, taskID)
	pipe.SRem(ctx, tm.keys.TaskCompleted, taskID)
	pipe.SRem(ctx, tm.keys.TaskFailed, taskID)

	_, err := pipe.Exec(ctx)
	return err
}

// ListTasks 列出任务
func (tm *TaskManager) ListTasks(ctx context.Context, filter *TaskFilter) ([]*Task, error) {
	var taskIDs []string
	var err error

	if filter == nil {
		filter = &TaskFilter{}
	}

	// 根据过滤条件获取任务ID列表
	if len(filter.Status) > 0 {
		taskIDs, err = tm.getTaskIDsByStatus(ctx, filter.Status, filter.Limit, filter.Offset)
	} else {
		taskIDs, err = tm.getAllTaskIDs(ctx, filter.Limit, filter.Offset)
	}

	if err != nil {
		return nil, err
	}

	// 批量获取任务详情
	tasks := make([]*Task, 0, len(taskIDs))
	for _, taskID := range taskIDs {
		task, err := tm.GetTask(ctx, taskID)
		if err != nil {
			continue // 忽略获取失败的任务
		}

		// 应用其他过滤条件
		if tm.matchFilter(task, filter) {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// PopTask 弹出下一个待处理任务 (优先级队列)
func (tm *TaskManager) PopTask(ctx context.Context) (*Task, error) {
	// 使用Lua脚本原子性地弹出任务
	script := `
		-- 从优先级队列弹出分数最高的任务
		local taskId = redis.call('ZPOPMAX', KEYS[1])
		if not taskId[1] then
			-- 如果优先级队列为空，从普通队列弹出
			taskId = redis.call('RPOP', KEYS[2])
			if not taskId then
				return nil
			end
			return taskId
		end
		return taskId[1]
	`

	result, err := tm.client.Eval(ctx, script, []string{tm.keys.TaskPriorityQueue, tm.keys.TaskQueue}).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to pop task: %w", err)
	}

	if result == nil {
		return nil, nil // 没有待处理的任务
	}

	taskID := result.(string)
	return tm.GetTask(ctx, taskID)
}

// PopTaskBatch 批量弹出任务
func (tm *TaskManager) PopTaskBatch(ctx context.Context, count int) ([]*Task, error) {
	if count <= 0 {
		return nil, fmt.Errorf("count must be positive")
	}

	tasks := make([]*Task, 0, count)

	for i := 0; i < count; i++ {
		task, err := tm.PopTask(ctx)
		if err != nil {
			return tasks, err
		}
		if task == nil {
			break // 没有更多任务
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// UpdateTaskStatus 更新任务状态
func (tm *TaskManager) UpdateTaskStatus(ctx context.Context, taskID string, status TaskStatus) error {
	task, err := tm.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	oldStatus := task.Status
	task.Status = status
	task.UpdatedAt = time.Now().Unix()
	task.Version++

	pipe := tm.client.Pipeline()

	// 更新任务数据
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}
	pipe.HSet(ctx, tm.keys.TaskHash, taskID, taskJSON)

	// 从旧状态集合中移除
	switch oldStatus {
	case TaskStatusScheduled:
		pipe.SRem(ctx, tm.keys.TaskScheduled, taskID)
	case TaskStatusRunning:
		pipe.SRem(ctx, tm.keys.TaskRunning, taskID)
	case TaskStatusCompleted:
		pipe.SRem(ctx, tm.keys.TaskCompleted, taskID)
	case TaskStatusFailed:
		pipe.SRem(ctx, tm.keys.TaskFailed, taskID)
	}

	// 添加到新状态集合
	switch status {
	case TaskStatusScheduled:
		pipe.SAdd(ctx, tm.keys.TaskScheduled, taskID)
	case TaskStatusRunning:
		pipe.SAdd(ctx, tm.keys.TaskRunning, taskID)
	case TaskStatusCompleted:
		pipe.SAdd(ctx, tm.keys.TaskCompleted, taskID)
	case TaskStatusFailed:
		pipe.SAdd(ctx, tm.keys.TaskFailed, taskID)
	}

	_, err = pipe.Exec(ctx)
	return err
}

// GetTasksByStatus 根据状态获取任务
func (tm *TaskManager) GetTasksByStatus(ctx context.Context, status TaskStatus, limit int) ([]*Task, error) {
	var setKey string

	switch status {
	case TaskStatusScheduled:
		setKey = tm.keys.TaskScheduled
	case TaskStatusRunning:
		setKey = tm.keys.TaskRunning
	case TaskStatusCompleted:
		setKey = tm.keys.TaskCompleted
	case TaskStatusFailed:
		setKey = tm.keys.TaskFailed
	default:
		return nil, fmt.Errorf("unsupported status for set query: %s", status)
	}

	// 获取指定数量的任务ID
	taskIDs, err := tm.client.SRandMemberN(ctx, setKey, int64(limit)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get task IDs: %w", err)
	}

	// 批量获取任务详情
	tasks := make([]*Task, 0, len(taskIDs))
	for _, taskID := range taskIDs {
		task, err := tm.GetTask(ctx, taskID)
		if err != nil {
			continue // 忽略获取失败的任务
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetPendingTaskCount 获取待处理任务数量
func (tm *TaskManager) GetPendingTaskCount(ctx context.Context) (int64, error) {
	pipe := tm.client.Pipeline()
	pipe.ZCard(ctx, tm.keys.TaskPriorityQueue)
	pipe.LLen(ctx, tm.keys.TaskQueue)

	results, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return results[0].(*redis.IntCmd).Val() + results[1].(*redis.IntCmd).Val(), nil
}

// GetRunningTaskCount 获取运行中任务数量
func (tm *TaskManager) GetRunningTaskCount(ctx context.Context) (int64, error) {
	return tm.client.SCard(ctx, tm.keys.TaskRunning).Result()
}

// GetCompletedTaskCount 获取已完成任务数量
func (tm *TaskManager) GetCompletedTaskCount(ctx context.Context) (int64, error) {
	return tm.client.SCard(ctx, tm.keys.TaskCompleted).Result()
}

// GetFailedTaskCount 获取失败任务数量
func (tm *TaskManager) GetFailedTaskCount(ctx context.Context) (int64, error) {
	return tm.client.SCard(ctx, tm.keys.TaskFailed).Result()
}

// CleanupExpiredTasks 清理过期任务
func (tm *TaskManager) CleanupExpiredTasks(ctx context.Context, expiredBefore int64) (int64, error) {
	// 使用Lua脚本原子性地清理过期任务
	script := `
		local expiredBefore = tonumber(ARGV[1])
		local taskHash = KEYS[1]
		local completedSet = KEYS[2]
		local failedSet = KEYS[3]
		
		local cleaned = 0
		
		-- 清理已完成的过期任务
		local completedTasks = redis.call('SMEMBERS', completedSet)
		for _, taskId in ipairs(completedTasks) do
			local taskJson = redis.call('HGET', taskHash, taskId)
			if taskJson then
				local task = cjson.decode(taskJson)
				if task.completedAt and task.completedAt < expiredBefore then
					redis.call('HDEL', taskHash, taskId)
					redis.call('SREM', completedSet, taskId)
					cleaned = cleaned + 1
				end
			end
		end
		
		-- 清理失败的过期任务
		local failedTasks = redis.call('SMEMBERS', failedSet)
		for _, taskId in ipairs(failedTasks) do
			local taskJson = redis.call('HGET', taskHash, taskId)
			if taskJson then
				local task = cjson.decode(taskJson)
				if task.updatedAt and task.updatedAt < expiredBefore then
					redis.call('HDEL', taskHash, taskId)
					redis.call('SREM', failedSet, taskId)
					cleaned = cleaned + 1
				end
			end
		end
		
		return cleaned
	`

	result, err := tm.client.Eval(ctx, script, []string{
		tm.keys.TaskHash,
		tm.keys.TaskCompleted,
		tm.keys.TaskFailed,
	}, expiredBefore).Result()

	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired tasks: %w", err)
	}

	return result.(int64), nil
}

// calculatePriorityScore 计算优先级分数
// 分数 = 优先级权重 * 1000000 + (当前时间 - 创建时间)
// 这样高优先级任务会有更高的分数，同优先级任务按创建时间排序
func (tm *TaskManager) calculatePriorityScore(priority TaskPriority, createdAt int64) float64 {
	var priorityWeight float64

	switch priority {
	case TaskPriorityCritical:
		priorityWeight = 4
	case TaskPriorityHigh:
		priorityWeight = 3
	case TaskPriorityNormal:
		priorityWeight = 2
	case TaskPriorityLow:
		priorityWeight = 1
	default:
		priorityWeight = 2
	}

	// 使用当前时间和创建时间的差值作为时间权重
	timeWeight := float64(time.Now().Unix() - createdAt)

	return priorityWeight*1000000 + timeWeight
}

// getTaskIDsByStatus 根据状态获取任务ID列表
func (tm *TaskManager) getTaskIDsByStatus(ctx context.Context, statuses []TaskStatus, limit, offset int) ([]string, error) {
	var allIDs []string

	for _, status := range statuses {
		var setKey string

		switch status {
		case TaskStatusPending:
			// 对于待处理任务，从队列中获取
			queueIDs, err := tm.client.LRange(ctx, tm.keys.TaskQueue, int64(offset), int64(offset+limit-1)).Result()
			if err == nil {
				allIDs = append(allIDs, queueIDs...)
			}

			// 同时从优先级队列获取
			priorityIDs, err := tm.client.ZRevRange(ctx, tm.keys.TaskPriorityQueue, int64(offset), int64(offset+limit-1)).Result()
			if err == nil {
				allIDs = append(allIDs, priorityIDs...)
			}

		case TaskStatusScheduled:
			setKey = tm.keys.TaskScheduled
		case TaskStatusRunning:
			setKey = tm.keys.TaskRunning
		case TaskStatusCompleted:
			setKey = tm.keys.TaskCompleted
		case TaskStatusFailed:
			setKey = tm.keys.TaskFailed
		}

		if setKey != "" {
			ids, err := tm.client.SMembers(ctx, setKey).Result()
			if err != nil {
				continue
			}
			allIDs = append(allIDs, ids...)
		}
	}

	// 去重并应用分页
	uniqueIDs := tm.removeDuplicates(allIDs)

	if offset >= len(uniqueIDs) {
		return []string{}, nil
	}

	end := offset + limit
	if end > len(uniqueIDs) {
		end = len(uniqueIDs)
	}

	return uniqueIDs[offset:end], nil
}

// getAllTaskIDs 获取所有任务ID
func (tm *TaskManager) getAllTaskIDs(ctx context.Context, limit, offset int) ([]string, error) {
	// 从任务哈希表获取所有字段名（任务ID）
	taskIDs, err := tm.client.HKeys(ctx, tm.keys.TaskHash).Result()
	if err != nil {
		return nil, err
	}

	// 应用分页
	if offset >= len(taskIDs) {
		return []string{}, nil
	}

	end := offset + limit
	if end > len(taskIDs) {
		end = len(taskIDs)
	}

	return taskIDs[offset:end], nil
}

// matchFilter 检查任务是否匹配过滤条件
func (tm *TaskManager) matchFilter(task *Task, filter *TaskFilter) bool {
	// 优先级过滤
	if len(filter.Priority) > 0 {
		found := false
		for _, p := range filter.Priority {
			if task.Priority == p {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 工作节点过滤
	if filter.WorkerID != "" && task.WorkerID != filter.WorkerID {
		return false
	}

	// 创建时间过滤
	if filter.CreatedAfter > 0 && task.CreatedAt <= filter.CreatedAfter {
		return false
	}

	if filter.CreatedBefore > 0 && task.CreatedAt >= filter.CreatedBefore {
		return false
	}

	// 标签过滤
	if len(filter.Tags) > 0 {
		taskTags := make(map[string]bool)
		for key := range task.Metadata {
			taskTags[key] = true
		}

		for _, tag := range filter.Tags {
			if !taskTags[tag] {
				return false
			}
		}
	}

	return true
}

// removeDuplicates 去除重复的任务ID
func (tm *TaskManager) removeDuplicates(ids []string) []string {
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

// FindOrphanTasks 查找孤儿任务（分配给已下线工作节点的任务）
func (tm *TaskManager) FindOrphanTasks(ctx context.Context, offlineWorkerIDs []string) ([]*Task, error) {
	if len(offlineWorkerIDs) == 0 {
		return nil, nil
	}

	// 创建离线工作节点ID的映射以便快速查找
	offlineWorkers := make(map[string]bool)
	for _, workerID := range offlineWorkerIDs {
		offlineWorkers[workerID] = true
	}

	// 获取所有运行中和已调度的任务
	var orphanTasks []*Task

	// 检查运行中的任务
	runningTaskIDs, err := tm.client.SMembers(ctx, tm.keys.TaskRunning).Result()
	if err == nil {
		for _, taskID := range runningTaskIDs {
			task, err := tm.GetTask(ctx, taskID)
			if err != nil {
				continue
			}
			if task.WorkerID != "" && offlineWorkers[task.WorkerID] {
				orphanTasks = append(orphanTasks, task)
			}
		}
	}

	// 检查已调度的任务
	scheduledTaskIDs, err := tm.client.SMembers(ctx, tm.keys.TaskScheduled).Result()
	if err == nil {
		for _, taskID := range scheduledTaskIDs {
			task, err := tm.GetTask(ctx, taskID)
			if err != nil {
				continue
			}
			if task.WorkerID != "" && offlineWorkers[task.WorkerID] {
				orphanTasks = append(orphanTasks, task)
			}
		}
	}

	return orphanTasks, nil
}

// RescheduleOrphanTasks 重调度孤儿任务
func (tm *TaskManager) RescheduleOrphanTasks(ctx context.Context, orphanTasks []*Task) error {
	if len(orphanTasks) == 0 {
		return nil
	}

	pipe := tm.client.Pipeline()

	for _, task := range orphanTasks {
		// 重置任务状态为待处理
		oldStatus := task.Status
		task.Status = TaskStatusPending
		task.WorkerID = ""
		task.ScheduledAt = 0
		task.UpdatedAt = time.Now().Unix()
		task.Version++

		// 序列化任务
		taskJSON, err := json.Marshal(task)
		if err != nil {
			continue
		}

		// 更新任务数据
		pipe.HSet(ctx, tm.keys.TaskHash, task.ID, taskJSON)

		// 从旧状态集合中移除
		switch oldStatus {
		case TaskStatusScheduled:
			pipe.SRem(ctx, tm.keys.TaskScheduled, task.ID)
		case TaskStatusRunning:
			pipe.SRem(ctx, tm.keys.TaskRunning, task.ID)
		}

		// 重新加入待处理队列
		score := tm.calculatePriorityScore(task.Priority, task.CreatedAt)
		pipe.ZAdd(ctx, tm.keys.TaskPriorityQueue, redis.Z{
			Score:  score,
			Member: task.ID,
		})
		pipe.LPush(ctx, tm.keys.TaskQueue, task.ID)
	}

	_, err := pipe.Exec(ctx)
	return err
}
