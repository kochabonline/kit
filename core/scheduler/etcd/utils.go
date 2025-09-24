package etcd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"runtime"
	"time"

	"github.com/rs/xid"
)

// generateTaskID 生成任务唯一标识符
func generateTaskID() string {
	return fmt.Sprintf("task-%s", xid.New().String())
}

// generateWorkerID 生成工作节点唯一标识符
func generateWorkerID() string {
	return fmt.Sprintf("worker-%s", xid.New().String())
}

// generateNodeID 生成调度器节点唯一标识符
func generateNodeID() string {
	return fmt.Sprintf("scheduler-%s", xid.New().String())
}

// generateSessionID 生成会话唯一标识符
func generateSessionID() string {
	return fmt.Sprintf("session-%s", xid.New().String())
}

// generateRandomString 生成指定长度的随机字符串
func generateRandomString(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		// 如果随机数生成失败，使用时间戳作为后备方案
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// currentTimestamp 获取当前时间戳（毫秒）
func currentTimestamp() int64 {
	return time.Now().UnixMilli()
}

// GetMemoryUsage 获取内存使用情况
func GetMemoryUsage() (int64, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.Alloc), nil
}

// GetGoroutineCount 获取Goroutine数量
func GetGoroutineCount() int {
	return runtime.NumGoroutine()
}

// formatDuration 格式化时间间隔
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.2fm", d.Minutes())
	}
	return fmt.Sprintf("%.2fh", d.Hours())
}

// validateTaskPriority 验证任务优先级
func validateTaskPriority(priority TaskPriority) bool {
	switch priority {
	case TaskPriorityLow, TaskPriorityNormal, TaskPriorityHigh, TaskPriorityCritical:
		return true
	default:
		return false
	}
}

// validateWorkerStatus 验证工作节点状态
func validateWorkerStatus(status WorkerStatus) bool {
	switch status {
	case WorkerStatusOnline, WorkerStatusOffline, WorkerStatusBusy, WorkerStatusIdle:
		return true
	default:
		return false
	}
}

// calculateLoadRatio 计算负载比例
func calculateLoadRatio(currentTasks int, maxConcurrency int) float64 {
	if maxConcurrency <= 0 {
		return 1.0
	}
	ratio := float64(currentTasks) / float64(maxConcurrency)
	if ratio > 1.0 {
		return 1.0
	}
	return ratio
}

// isTaskExecutable 检查任务是否可执行
func isTaskExecutable(task *Task) bool {
	switch task.Status {
	case TaskStatusPending, TaskStatusRetrying:
		return true
	default:
		return false
	}
}

// isWorkerAvailable 检查工作节点是否可用
func isWorkerAvailable(worker *Worker) bool {
	switch worker.Status {
	case WorkerStatusOnline, WorkerStatusIdle:
		return worker.RunningTaskNum < worker.MaxConcurrency
	default:
		return false
	}
}

// copyTask 创建任务的深拷贝
func copyTask(task *Task) *Task {
	if task == nil {
		return nil
	}

	taskCopy := *task

	// 深拷贝map字段
	if task.Payload != nil {
		taskCopy.Payload = make(map[string]any)
		for k, v := range task.Payload {
			taskCopy.Payload[k] = v
		}
	}

	if task.Metadata != nil {
		taskCopy.Metadata = make(map[string]string)
		for k, v := range task.Metadata {
			taskCopy.Metadata[k] = v
		}
	}

	if task.Dependencies != nil {
		taskCopy.Dependencies = make([]string, len(task.Dependencies))
		copy(taskCopy.Dependencies, task.Dependencies)
	}

	return &taskCopy
}

// copyWorker 创建工作节点的深拷贝
func copyWorker(worker *Worker) *Worker {
	if worker == nil {
		return nil
	}

	workerCopy := *worker

	// 深拷贝slice和map字段
	if worker.Capabilities != nil {
		workerCopy.Capabilities = make([]string, len(worker.Capabilities))
		copy(workerCopy.Capabilities, worker.Capabilities)
	}

	if worker.CurrentTasks != nil {
		workerCopy.CurrentTasks = make([]string, len(worker.CurrentTasks))
		copy(workerCopy.CurrentTasks, worker.CurrentTasks)
	}

	if worker.Tags != nil {
		workerCopy.Tags = make(map[string]string)
		for k, v := range worker.Tags {
			workerCopy.Tags[k] = v
		}
	}

	if worker.Metadata != nil {
		workerCopy.Metadata = make(map[string]string)
		for k, v := range worker.Metadata {
			workerCopy.Metadata[k] = v
		}
	}

	return &workerCopy
}

// mergeWorkerTags 合并工作节点标签
func mergeWorkerTags(base, override map[string]string) map[string]string {
	result := make(map[string]string)

	// 复制基础标签
	for k, v := range base {
		result[k] = v
	}

	// 覆盖标签
	for k, v := range override {
		result[k] = v
	}

	return result
}

// sanitizeString 清理字符串，移除不安全字符
func sanitizeString(s string) string {
	// 简单的实现，可以根据需要扩展
	if len(s) > 255 {
		s = s[:255]
	}
	return s
}

// calculateRetryBackoff 计算重试退避时间
func calculateRetryBackoff(retryCount int, baseDelay, maxDelay time.Duration) time.Duration {
	delay := baseDelay
	for i := 0; i < retryCount; i++ {
		delay *= 2
		if delay > maxDelay {
			return maxDelay
		}
	}
	return delay
}

// timeInRange 检查时间戳是否在指定范围内（毫秒时间戳）
func timestampInRange(timestamp, start, end int64) bool {
	if start == 0 && end == 0 {
		return true
	}
	if start == 0 {
		return timestamp <= end
	}
	if end == 0 {
		return timestamp >= start
	}
	return timestamp >= start && timestamp <= end
}

// timeInRange 检查时间是否在指定范围内（保留以兼容现有代码）
func timeInRange(t time.Time, start, end time.Time) bool {
	if start.IsZero() && end.IsZero() {
		return true
	}
	if start.IsZero() {
		return t.Before(end) || t.Equal(end)
	}
	if end.IsZero() {
		return t.After(start) || t.Equal(start)
	}
	return (t.After(start) || t.Equal(start)) && (t.Before(end) || t.Equal(end))
}

// floatInRange 检查浮点数是否在指定范围内
func floatInRange(value, min, max float64) bool {
	return value >= min && value <= max
}

// StringSliceContains 检查字符串切片是否包含指定元素
func StringSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// RemoveStringFromSlice 从字符串切片中移除指定元素
func RemoveStringFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

// MapStringStringCopy 复制字符串到字符串的映射
func MapStringStringCopy(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	result := make(map[string]string, len(source))
	for k, v := range source {
		result[k] = v
	}
	return result
}

// MapStringAnyCopy 复制字符串到任意类型的映射
func MapStringAnyCopy(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	result := make(map[string]any, len(source))
	for k, v := range source {
		result[k] = v
	}
	return result
}
