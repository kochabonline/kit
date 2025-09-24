package redis

import (
	"context"
	"time"
)

// Scheduler Redis分布式任务调度器接口
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

// EventCallback 事件回调函数类型
type EventCallback func(ctx context.Context, event *SchedulerEvent) error

// EventListener 事件监听器类型
type EventListener func(event *SchedulerEvent)

// DistributedLock 分布式锁接口 - Redis实现的分布式锁
type DistributedLock interface {
	// TryLock 尝试获取锁，非阻塞
	TryLock(ctx context.Context, key string, value string, ttl time.Duration) (bool, error)

	// Lock 获取锁，阻塞直到获取成功或超时
	Lock(ctx context.Context, key string, value string, ttl time.Duration, timeout time.Duration) error

	// Unlock 释放锁
	Unlock(ctx context.Context, key string, value string) error

	// Refresh 续期锁
	Refresh(ctx context.Context, key string, value string, ttl time.Duration) error
}

// LeaderElector Leader选举器接口 - 基于Redis分布式锁实现
type LeaderElector interface {
	// CampaignLeader 竞选Leader
	CampaignLeader(ctx context.Context) error

	// IsLeader 检查是否为Leader
	IsLeader() bool

	// ResignLeader 主动放弃Leader
	ResignLeader(ctx context.Context) error

	// GetLeaderID 获取当前Leader ID
	GetLeaderID(ctx context.Context) (string, error)

	// WatchLeaderElection 监听Leader选举事件
	WatchLeaderElection(ctx context.Context) <-chan LeaderElectionEvent
}

// PubSubManager Redis Pub/Sub管理器接口
type PubSubManager interface {
	// Publish 发布消息
	Publish(ctx context.Context, channel string, message interface{}) error

	// Subscribe 订阅消息
	Subscribe(ctx context.Context, channels ...string) <-chan *Message

	// Unsubscribe 取消订阅
	Unsubscribe(ctx context.Context, channels ...string) error

	// Close 关闭连接
	Close() error
}

// StreamManager Redis Stream管理器接口
type StreamManager interface {
	// AddMessage 添加消息到Stream
	AddMessage(ctx context.Context, stream string, values map[string]interface{}) (string, error)

	// ReadGroup 消费者组读取消息
	ReadGroup(ctx context.Context, group, consumer, stream, lastID string, count int64) ([]StreamMessage, error)

	// CreateGroup 创建消费者组
	CreateGroup(ctx context.Context, stream, group, startID string) error

	// AckMessage 确认消息
	AckMessage(ctx context.Context, stream, group string, messageIDs ...string) error
}
