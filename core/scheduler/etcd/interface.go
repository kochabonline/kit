package etcd

import (
	"context"
)

// Scheduler 事件驱动分布式任务调度器接口
type Scheduler interface {
	// Start 启动调度器
	Start(ctx context.Context) error

	// Stop 停止调度器
	Stop(ctx context.Context) error

	// SubmitTask 提交任务
	SubmitTask(ctx context.Context, task *Task) error

	// GetTask 获取任务信息
	GetTask(ctx context.Context, taskID string) (*Task, error)

	// UpdateTask 更新任务信息
	UpdateTask(ctx context.Context, task *Task) error

	// CancelTask 取消任务
	CancelTask(ctx context.Context, taskID string) error

	// ListTasks 列出任务
	ListTasks(ctx context.Context, filter *TaskFilter) ([]*Task, error)

	// RegisterWorker 注册工作节点
	RegisterWorker(ctx context.Context, worker *Worker) error

	// UnregisterWorker 注销工作节点
	UnregisterWorker(ctx context.Context, workerID string) error

	// GetWorker 获取工作节点信息
	GetWorker(ctx context.Context, workerID string) (*Worker, error)

	// ListWorkers 列出工作节点
	ListWorkers(ctx context.Context, filter *WorkerFilter) ([]*Worker, error)

	// RegisterEventCallback 注册事件回调
	RegisterEventCallback(eventType EventType, callback EventCallback) error

	// IsLeader 检查当前节点是否为Leader
	IsLeader() bool

	// GetNodeID 获取当前节点ID
	GetNodeID() string

	// GetMetrics 获取调度器指标
	GetMetrics() *SchedulerMetrics

	// GetLoadBalanceStrategy 获取当前负载均衡策略
	GetLoadBalanceStrategy() LoadBalanceStrategy

	// SetLoadBalanceStrategy 设置负载均衡策略
	SetLoadBalanceStrategy(strategy LoadBalanceStrategy)

	// ResetLoadBalancer 重置负载均衡器内部状态
	ResetLoadBalancer()

	// Health 健康检查
	Health(ctx context.Context) error
}

// TaskProcessor 统一的任务处理器接口
type TaskProcessor interface {
	Process(ctx context.Context, task *Task) error
}
