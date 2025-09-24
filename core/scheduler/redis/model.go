package redis

import (
	"errors"
	"sync"
	"time"
)

var (
	// ErrTaskNotFound 任务未找到错误
	ErrTaskNotFound = errors.New("task not found")
	// ErrWorkerNotFound 工作节点未找到错误
	ErrWorkerNotFound = errors.New("worker not found")
	// ErrTaskAlreadyExists 任务已存在错误
	ErrTaskAlreadyExists = errors.New("task already exists")
	// ErrWorkerAlreadyExists 工作节点已存在错误
	ErrWorkerAlreadyExists = errors.New("worker already exists")
	// ErrInvalidTaskStatus 无效任务状态错误
	ErrInvalidTaskStatus = errors.New("invalid task status")
	// ErrInvalidWorkerStatus 无效工作节点状态错误
	ErrInvalidWorkerStatus = errors.New("invalid worker status")
	// ErrSchedulerNotStarted 调度器未启动错误
	ErrSchedulerNotStarted = errors.New("scheduler not started")
	// ErrSchedulerAlreadyStarted 调度器已启动错误
	ErrSchedulerAlreadyStarted = errors.New("scheduler already started")
	// ErrNoAvailableWorkers 无可用工作节点错误
	ErrNoAvailableWorkers = errors.New("no available workers")
	// ErrTaskTimeout 任务超时错误
	ErrTaskTimeout = errors.New("task timeout")
	// ErrTaskCanceled 任务被取消错误
	ErrTaskCanceled = errors.New("task canceled")
	// ErrMaxRetriesExceeded 超过最大重试次数错误
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
	// ErrWorkerOffline 工作节点离线错误
	ErrWorkerOffline = errors.New("worker offline")
	// ErrContextCanceled 上下文取消错误
	ErrContextCanceled = errors.New("context canceled")
	// ErrLeaderElectionFailed 选举失败错误
	ErrLeaderElectionFailed = errors.New("leader election failed")
	// ErrLockAcquisitionFailed 获取分布式锁失败
	ErrLockAcquisitionFailed = errors.New("lock acquisition failed")
	// ErrLockAlreadyHeld 锁已被持有
	ErrLockAlreadyHeld = errors.New("lock already held")
	// ErrRedisConnectionFailed Redis连接失败
	ErrRedisConnectionFailed = errors.New("redis connection failed")
)

// TaskStatus 任务状态枚举
type TaskStatus int8

const (
	TaskStatusPending   TaskStatus = iota // 待执行
	TaskStatusScheduled                   // 已调度
	TaskStatusRunning                     // 执行中
	TaskStatusCompleted                   // 已完成
	TaskStatusFailed                      // 执行失败
	TaskStatusCanceled                    // 已取消
	TaskStatusRetrying                    // 重试中
)

// String 返回任务状态的字符串表示
func (ts TaskStatus) String() string {
	switch ts {
	case TaskStatusPending:
		return "pending"
	case TaskStatusScheduled:
		return "scheduled"
	case TaskStatusRunning:
		return "running"
	case TaskStatusCompleted:
		return "completed"
	case TaskStatusFailed:
		return "failed"
	case TaskStatusCanceled:
		return "canceled"
	case TaskStatusRetrying:
		return "retrying"
	default:
		return "unknown"
	}
}

// WorkerStatus 工作节点状态枚举
type WorkerStatus int8

const (
	WorkerStatusOnline  WorkerStatus = iota // 在线
	WorkerStatusOffline                     // 离线
	WorkerStatusBusy                        // 繁忙
	WorkerStatusIdle                        // 空闲
)

// String 返回工作节点状态的字符串表示
func (ws WorkerStatus) String() string {
	switch ws {
	case WorkerStatusOnline:
		return "online"
	case WorkerStatusOffline:
		return "offline"
	case WorkerStatusBusy:
		return "busy"
	case WorkerStatusIdle:
		return "idle"
	default:
		return "unknown"
	}
}

// TaskPriority 任务优先级
type TaskPriority int8

const (
	TaskPriorityLow      TaskPriority = iota // 低优先级
	TaskPriorityNormal                       // 普通优先级
	TaskPriorityHigh                         // 高优先级
	TaskPriorityCritical                     // 紧急优先级
)

// String 返回任务优先级的字符串表示
func (p TaskPriority) String() string {
	switch p {
	case TaskPriorityLow:
		return "low"
	case TaskPriorityNormal:
		return "normal"
	case TaskPriorityHigh:
		return "high"
	case TaskPriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// LoadBalanceStrategy 负载均衡策略
type LoadBalanceStrategy int8

const (
	StrategyLeastTasks         LoadBalanceStrategy = iota // 最少任务数
	StrategyRoundRobin                                    // 轮询
	StrategyWeightedRoundRobin                            // 加权轮询
	StrategyRandom                                        // 随机
	StrategyConsistentHash                                // 一致性哈希
	StrategyLeastConnections                              // 最少连接数
)

// EventType 事件类型
type EventType string

const (
	EventTaskSubmitted    EventType = "task_submitted"    // 任务提交
	EventTaskScheduled    EventType = "task_scheduled"    // 任务调度
	EventTaskStarted      EventType = "task_started"      // 任务开始
	EventTaskCompleted    EventType = "task_completed"    // 任务完成
	EventTaskFailed       EventType = "task_failed"       // 任务失败
	EventTaskCanceled     EventType = "task_canceled"     // 任务取消
	EventWorkerJoined     EventType = "worker_joined"     // 工作节点加入
	EventWorkerLeft       EventType = "worker_left"       // 工作节点离开
	EventWorkerOnline     EventType = "worker_online"     // 工作节点上线
	EventWorkerOffline    EventType = "worker_offline"    // 工作节点离线
	EventWorkerBusy       EventType = "worker_busy"       // 工作节点忙碌
	EventWorkerIdle       EventType = "worker_idle"       // 工作节点空闲
	EventLeaderElected    EventType = "leader_elected"    // Leader选举
	EventLeaderLost       EventType = "leader_lost"       // Leader丢失
	EventSchedulerStarted EventType = "scheduler_started" // 调度器启动
	EventSchedulerStopped EventType = "scheduler_stopped" // 调度器停止
)

// String 返回事件类型的字符串表示
func (et EventType) String() string {
	return string(et)
}

// Task 任务定义
type Task struct {
	// 基本信息
	ID       string            `json:"id"`                 // 任务唯一标识
	Name     string            `json:"name"`               // 任务名称
	Priority TaskPriority      `json:"priority"`           // 任务优先级
	Status   TaskStatus        `json:"status"`             // 任务状态
	Payload  map[string]any    `json:"payload,omitempty"`  // 任务参数
	Metadata map[string]string `json:"metadata,omitempty"` // 元数据

	// 调度信息
	ScheduledAt int64 `json:"scheduledAt,omitempty"` // 任务被调度的时间戳

	// 执行信息
	WorkerID    string `json:"workerId,omitempty"`    // 执行节点ID
	StartedAt   int64  `json:"startedAt,omitempty"`   // 开始执行时间戳
	CompletedAt int64  `json:"completedAt,omitempty"` // 完成时间戳
	Error       string `json:"error,omitempty"`       // 错误信息

	// 重试配置
	RetryCount    int           `json:"retryCount"`              // 已重试次数
	MaxRetries    int           `json:"maxRetries"`              // 最大重试次数
	RetryInterval time.Duration `json:"retryInterval,omitempty"` // 重试间隔

	// 超时配置
	Timeout        time.Duration `json:"timeout,omitempty"` // 执行超时时间
	HeartbeatCheck bool          `json:"heartbeatCheck"`    // 是否启用心跳检查

	// 依赖关系
	Dependencies []string `json:"dependencies,omitempty"` // 依赖的任务ID列表

	// 时间戳
	CreatedAt int64 `json:"createdAt"` // 创建时间戳
	UpdatedAt int64 `json:"updatedAt"` // 更新时间戳

	// 版本控制 - 用于并发更新检查
	Version int64 `json:"version"` // 版本号
}

// Worker 工作节点定义
type Worker struct {
	// 基本信息
	ID     string       `json:"id"`     // 节点唯一标识
	Name   string       `json:"name"`   // 节点名称
	Status WorkerStatus `json:"status"` // 节点状态
	Tags   []string     `json:"tags"`   // 节点标签

	// 网络信息
	Address string            `json:"address"`           // 节点地址
	Port    int               `json:"port"`              // 端口号
	Headers map[string]string `json:"headers,omitempty"` // 请求头

	// 能力信息
	Capabilities []string `json:"capabilities"` // 支持的任务类型
	MaxTasks     int      `json:"maxTasks"`     // 最大并发任务数
	CurrentTasks int      `json:"currentTasks"` // 当前任务数

	// 资源信息
	CPU    float64 `json:"cpu"`    // CPU使用率
	Memory float64 `json:"memory"` // 内存使用率
	Load   float64 `json:"load"`   // 负载

	// 权重和优先级
	Weight   int `json:"weight"`   // 权重
	Priority int `json:"priority"` // 优先级

	// 时间信息
	LastHeartbeat int64 `json:"lastHeartbeat"` // 最后心跳时间戳
	CreatedAt     int64 `json:"createdAt"`     // 创建时间戳
	UpdatedAt     int64 `json:"updatedAt"`     // 更新时间戳

	// 统计信息
	CompletedTasks int64 `json:"completedTasks"` // 已完成任务数
	FailedTasks    int64 `json:"failedTasks"`    // 失败任务数

	// 版本控制
	Version int64 `json:"version"` // 版本号
}

// SchedulerEvent 调度器事件
type SchedulerEvent struct {
	Type      EventType              `json:"type"`                // 事件类型
	TaskID    string                 `json:"taskId,omitempty"`    // 任务ID
	WorkerID  string                 `json:"workerId,omitempty"`  // 工作节点ID
	NodeID    string                 `json:"nodeId,omitempty"`    // 调度器节点ID
	Timestamp int64                  `json:"timestamp"`           // 事件时间戳
	Data      map[string]interface{} `json:"data,omitempty"`      // 事件数据
	Error     string                 `json:"error,omitempty"`     // 错误信息
	Retryable bool                   `json:"retryable,omitempty"` // 是否可重试
}

// SchedulerMetrics 调度器指标
type SchedulerMetrics struct {
	// 基本指标
	NodeID           string `json:"nodeId"`           // 节点ID
	IsLeader         bool   `json:"isLeader"`         // 是否为Leader
	StartTime        int64  `json:"startTime"`        // 启动时间
	Uptime           int64  `json:"uptime"`           // 运行时长
	LastLeaderChange int64  `json:"lastLeaderChange"` // 最后一次Leader变更时间
	LeaderTerm       int64  `json:"leaderTerm"`       // Leader任期

	// 任务指标
	TasksSubmitted  int64 `json:"tasksSubmitted"`  // 已提交任务数
	TasksScheduled  int64 `json:"tasksScheduled"`  // 已调度任务数
	TasksCompleted  int64 `json:"tasksCompleted"`  // 已完成任务数
	TasksFailed     int64 `json:"tasksFailed"`     // 失败任务数
	TasksCanceled   int64 `json:"tasksCanceled"`   // 已取消任务数
	TasksRetrying   int64 `json:"tasksRetrying"`   // 重试中任务数
	PendingTasks    int64 `json:"pendingTasks"`    // 待处理任务数
	RunningTasks    int64 `json:"runningTasks"`    // 运行中任务数
	AvgTaskDuration int64 `json:"avgTaskDuration"` // 平均任务执行时长

	// 工作节点指标
	OnlineWorkers  int64 `json:"onlineWorkers"`  // 在线工作节点数
	OfflineWorkers int64 `json:"offlineWorkers"` // 离线工作节点数
	BusyWorkers    int64 `json:"busyWorkers"`    // 繁忙工作节点数
	IdleWorkers    int64 `json:"idleWorkers"`    // 空闲工作节点数

	// 性能指标
	SchedulingLatency   int64   `json:"schedulingLatency"`   // 调度延迟(毫秒)
	ThroughputPerMin    float64 `json:"throughputPerMin"`    // 每分钟吞吐量
	ErrorRate           float64 `json:"errorRate"`           // 错误率
	ResourceUtilization float64 `json:"resourceUtilization"` // 资源利用率

	// 负载均衡指标
	LoadBalanceStrategy   string  `json:"loadBalanceStrategy"`   // 负载均衡策略
	LoadBalanceEfficiency float64 `json:"loadBalanceEfficiency"` // 负载均衡效率

	// Redis相关指标
	RedisConnections   int64 `json:"redisConnections"`   // Redis连接数
	RedisLatency       int64 `json:"redisLatency"`       // Redis延迟
	RedisPubSubLatency int64 `json:"redisPubSubLatency"` // Redis Pub/Sub延迟
	RedisStreamLag     int64 `json:"redisStreamLag"`     // Redis Stream消息延迟

	// 锁定相关
	mu sync.RWMutex
}

// TaskFilter 任务过滤器
type TaskFilter struct {
	Status        []TaskStatus   `json:"status,omitempty"`        // 状态过滤
	Priority      []TaskPriority `json:"priority,omitempty"`      // 优先级过滤
	WorkerID      string         `json:"workerId,omitempty"`      // 工作节点ID
	CreatedAfter  int64          `json:"createdAfter,omitempty"`  // 创建时间之后
	CreatedBefore int64          `json:"createdBefore,omitempty"` // 创建时间之前
	Tags          []string       `json:"tags,omitempty"`          // 标签过滤
	Limit         int            `json:"limit,omitempty"`         // 限制数量
	Offset        int            `json:"offset,omitempty"`        // 偏移量
}

// WorkerFilter 工作节点过滤器
type WorkerFilter struct {
	Status       []WorkerStatus `json:"status,omitempty"`       // 状态过滤
	Capabilities []string       `json:"capabilities,omitempty"` // 能力过滤
	Tags         []string       `json:"tags,omitempty"`         // 标签过滤
	MinLoad      float64        `json:"minLoad,omitempty"`      // 最小负载
	MaxLoad      float64        `json:"maxLoad,omitempty"`      // 最大负载
	OnlineOnly   bool           `json:"onlineOnly,omitempty"`   // 仅在线节点
	Limit        int            `json:"limit,omitempty"`        // 限制数量
	Offset       int            `json:"offset,omitempty"`       // 偏移量
}

// SchedulerOptions 调度器配置选项
type SchedulerOptions struct {
	// 基本配置
	NodeID        string `yaml:"nodeId"`        // 节点ID
	RedisAddr     string `yaml:"redisAddr"`     // Redis地址
	RedisPassword string `yaml:"redisPassword"` // Redis密码
	RedisDB       int    `yaml:"redisDB"`       // Redis数据库

	// 选举配置
	ElectionTimeout     time.Duration `yaml:"electionTimeout"`     // 选举超时时间
	LeaderLeaseDuration time.Duration `yaml:"leaderLeaseDuration"` // Leader租约时长
	HeartbeatInterval   time.Duration `yaml:"heartbeatInterval"`   // 心跳间隔

	// 任务配置
	TaskTimeout       time.Duration `yaml:"taskTimeout"`       // 任务超时时间
	TaskRetryInterval time.Duration `yaml:"taskRetryInterval"` // 任务重试间隔
	MaxRetries        int           `yaml:"maxRetries"`        // 最大重试次数
	EnableHeartbeat   bool          `yaml:"enableHeartbeat"`   // 启用心跳检查

	// 负载均衡配置
	LoadBalanceStrategy LoadBalanceStrategy `yaml:"loadBalanceStrategy"` // 负载均衡策略

	// 监控配置
	EnableMetrics       bool          `yaml:"enableMetrics"`       // 启用指标收集
	MetricsInterval     time.Duration `yaml:"metricsInterval"`     // 指标收集间隔
	HealthCheckInterval time.Duration `yaml:"healthCheckInterval"` // 健康检查间隔

	// 事件配置
	EventBufferSize int           `yaml:"eventBufferSize"` // 事件缓冲区大小
	EventTimeout    time.Duration `yaml:"eventTimeout"`    // 事件超时时间

	// Redis配置
	RedisPoolSize   int           `yaml:"redisPoolSize"`   // Redis连接池大小
	RedisTimeout    time.Duration `yaml:"redisTimeout"`    // Redis操作超时
	RedisRetryCount int           `yaml:"redisRetryCount"` // Redis重试次数
	RedisRetryDelay time.Duration `yaml:"redisRetryDelay"` // Redis重试延迟

	// Stream配置
	StreamMaxLen     int64  `yaml:"streamMaxLen"`     // Stream最大长度
	StreamGroup      string `yaml:"streamGroup"`      // Stream消费者组
	StreamConsumerID string `yaml:"streamConsumerID"` // Stream消费者ID
}

// DefaultSchedulerOptions 返回默认的调度器配置
func DefaultSchedulerOptions() *SchedulerOptions {
	return &SchedulerOptions{
		NodeID:              "",
		RedisAddr:           "localhost:6379",
		RedisPassword:       "",
		RedisDB:             0,
		ElectionTimeout:     30 * time.Second,
		LeaderLeaseDuration: 60 * time.Second,
		HeartbeatInterval:   10 * time.Second,
		TaskTimeout:         5 * time.Minute,
		TaskRetryInterval:   30 * time.Second,
		MaxRetries:          3,
		EnableHeartbeat:     true,
		LoadBalanceStrategy: StrategyLeastTasks,
		EnableMetrics:       true,
		MetricsInterval:     30 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		EventBufferSize:     1000,
		EventTimeout:        10 * time.Second,
		RedisPoolSize:       10,
		RedisTimeout:        5 * time.Second,
		RedisRetryCount:     3,
		RedisRetryDelay:     100 * time.Millisecond,
		StreamMaxLen:        10000,
		StreamGroup:         "scheduler-group",
		StreamConsumerID:    "",
	}
}

// Message Redis消息结构
type Message struct {
	Channel string `json:"channel"` // 频道
	Pattern string `json:"pattern"` // 模式
	Payload string `json:"payload"` // 消息内容
}

// StreamMessage Redis Stream消息结构
type StreamMessage struct {
	ID     string            `json:"id"`     // 消息ID
	Values map[string]string `json:"values"` // 消息内容
}

// LeaderElectionEvent Leader选举事件
type LeaderElectionEvent struct {
	Type      string `json:"type"`      // 事件类型: elected, lost, campaigning
	LeaderID  string `json:"leaderId"`  // Leader ID
	NodeID    string `json:"nodeId"`    // 当前节点ID
	Timestamp int64  `json:"timestamp"` // 时间戳
}

// RedisKeys Redis键名常量
type RedisKeys struct {
	// 分布式锁相关
	LeaderLock string // Leader锁
	NodeLock   string // 节点锁

	// 任务相关
	TaskHash          string // 任务哈希表
	TaskQueue         string // 任务队列
	TaskPriorityQueue string // 优先级队列
	TaskScheduled     string // 已调度任务集合
	TaskRunning       string // 运行中任务集合
	TaskCompleted     string // 已完成任务集合
	TaskFailed        string // 失败任务集合

	// 工作节点相关
	WorkerHash      string // 工作节点哈希表
	WorkerOnline    string // 在线工作节点集合
	WorkerBusy      string // 繁忙工作节点集合
	WorkerHeartbeat string // 工作节点心跳

	// 事件相关
	EventStream  string // 事件流
	EventChannel string // 事件频道

	// 指标相关
	MetricsHash string // 指标哈希表
}

// NewRedisKeys 创建Redis键名
func NewRedisKeys(prefix string) *RedisKeys {
	return &RedisKeys{
		LeaderLock:        prefix + ":lock:leader",
		NodeLock:          prefix + ":lock:node:",
		TaskHash:          prefix + ":task",
		TaskQueue:         prefix + ":queue:task",
		TaskPriorityQueue: prefix + ":queue:priority",
		TaskScheduled:     prefix + ":set:scheduled",
		TaskRunning:       prefix + ":set:running",
		TaskCompleted:     prefix + ":set:completed",
		TaskFailed:        prefix + ":set:failed",
		WorkerHash:        prefix + ":worker",
		WorkerOnline:      prefix + ":set:worker:online",
		WorkerBusy:        prefix + ":set:worker:busy",
		WorkerHeartbeat:   prefix + ":heartbeat:worker:",
		EventStream:       prefix + ":stream:event",
		EventChannel:      prefix + ":channel:event",
		MetricsHash:       prefix + ":metrics",
	}
}
