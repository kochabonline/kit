package etcd

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
}

// TaskFilter 任务过滤器
type TaskFilter struct {
	Status        []TaskStatus   `json:"status,omitempty"`        // 任务状态过滤
	WorkerID      string         `json:"workerId,omitempty"`      // 工作节点ID过滤
	Priority      []TaskPriority `json:"priority,omitempty"`      // 优先级过滤
	CreatedAfter  int64          `json:"createdAfter,omitempty"`  // 创建时间过滤(之后)
	CreatedBefore int64          `json:"createdBefore,omitempty"` // 创建时间过滤(之前)
	Limit         int            `json:"limit,omitempty"`         // 限制返回数量
	Offset        int            `json:"offset,omitempty"`        // 偏移量
}

// TaskExecutionResult 任务执行结果
type TaskExecutionResult struct {
	TaskID      string         `json:"taskId"`      // 任务ID
	Status      TaskStatus     `json:"status"`      // 执行状态
	Result      map[string]any `json:"result"`      // 执行结果
	Error       string         `json:"error"`       // 错误信息
	StartedAt   int64          `json:"startedAt"`   // 开始时间戳
	CompletedAt int64          `json:"completedAt"` // 完成时间戳
	Duration    time.Duration  `json:"duration"`    // 执行耗时
	Metrics     map[string]any `json:"metrics"`     // 执行指标
}

// Worker 工作节点定义
type Worker struct {
	// 基本信息
	ID           string            `json:"id"`                 // 节点唯一标识
	Name         string            `json:"name"`               // 节点名称
	Status       WorkerStatus      `json:"status"`             // 节点状态
	Version      string            `json:"version"`            // 节点版本
	Capabilities []string          `json:"capabilities"`       // 支持的任务类型
	Tags         map[string]string `json:"tags,omitempty"`     // 标签
	Metadata     map[string]string `json:"metadata,omitempty"` // 元数据

	// 负载信息
	MaxConcurrency int      `json:"maxConcurrency"` // 最大并发数
	CurrentTasks   []string `json:"currentTasks"`   // 当前执行的任务ID列表
	RunningTaskNum int      `json:"runningTaskNum"` // 当前运行任务数
	TotalProcessed int64    `json:"totalProcessed"` // 累计处理任务数
	TotalFailed    int64    `json:"totalFailed"`    // 累计失败任务数
	Weight         int      `json:"weight"`         // 权重
	LoadRatio      float64  `json:"loadRatio"`      // 负载比例

	// 健康状态
	LastHeartbeat int64         `json:"lastHeartbeat"`           // 最后心跳时间戳
	HealthStatus  HealthStatus  `json:"healthStatus"`            // 健康状态
	ResourceUsage ResourceUsage `json:"resourceUsage,omitempty"` // 资源使用情况

	// 时间戳
	RegisteredAt int64 `json:"registeredAt"` // 注册时间戳
	UpdatedAt    int64 `json:"updatedAt"`    // 更新时间戳
}

// WorkerFilter 工作节点过滤器
type WorkerFilter struct {
	Status       []WorkerStatus    `json:"status,omitempty"`       // 状态过滤
	Capabilities []string          `json:"capabilities,omitempty"` // 能力过滤
	Tags         map[string]string `json:"tags,omitempty"`         // 标签过滤
	Limit        int               `json:"limit,omitempty"`        // 限制返回数量
	Offset       int               `json:"offset,omitempty"`       // 偏移量
}

// SchedulerEvent 调度器事件
type SchedulerEvent struct {
	Type      EventType      `json:"type"`               // 事件类型
	TaskID    string         `json:"taskId,omitempty"`   // 任务ID
	WorkerID  string         `json:"workerId,omitempty"` // 工作节点ID
	Timestamp int64          `json:"timestamp"`          // 时间戳
	Data      map[string]any `json:"data,omitempty"`     // 事件数据
}

// SchedulerMetrics 调度器指标
type SchedulerMetrics struct {
	// 任务指标
	TasksTotal     int64 `json:"tasksTotal"`     // 总任务数
	TasksPending   int64 `json:"tasksPending"`   // 待处理任务数
	TasksRunning   int64 `json:"tasksRunning"`   // 运行中任务数
	TasksCompleted int64 `json:"tasksCompleted"` // 已完成任务数
	TasksFailed    int64 `json:"tasksFailed"`    // 失败任务数
	TasksCanceled  int64 `json:"tasksCanceled"`  // 取消任务数
	TasksRetrying  int64 `json:"tasksRetrying"`  // 重试中任务数

	// 工作节点指标
	WorkersTotal   int64 `json:"workersTotal"`   // 总工作节点数
	WorkersOnline  int64 `json:"workersOnline"`  // 在线工作节点数
	WorkersOffline int64 `json:"workersOffline"` // 离线工作节点数
	WorkersBusy    int64 `json:"workersBusy"`    // 忙碌工作节点数
	WorkersIdle    int64 `json:"workersIdle"`    // 空闲工作节点数

	// 性能指标
	AvgTaskDuration    time.Duration `json:"avgTaskDuration"`    // 平均任务执行时间
	TaskThroughput     float64       `json:"taskThroughput"`     // 任务吞吐量(任务/秒)
	SchedulerUptime    time.Duration `json:"schedulerUptime"`    // 调度器运行时间
	LastScheduleTime   int64         `json:"lastScheduleTime"`   // 最后调度时间
	TotalScheduleCount int64         `json:"totalScheduleCount"` // 总调度次数

	// 系统指标
	MemoryUsage    int64 `json:"memoryUsage"`    // 内存使用量(字节)
	GoroutineCount int   `json:"goroutineCount"` // Goroutine数量
	UpdatedAt      int64 `json:"updatedAt"`      // 指标更新时间

	// 内部同步
	mu sync.RWMutex `json:"-"`
}

// HealthStatus 健康状态
type HealthStatus struct {
	CPU    float64 `json:"cpu"`    // CPU使用率
	Memory float64 `json:"memory"` // 内存使用率
	Disk   float64 `json:"disk"`   // 磁盘使用率
	Load   float64 `json:"load"`   // 系统负载
}

// ResourceUsage 资源使用情况
type ResourceUsage struct {
	CPUPercent    float64 `json:"cpuPercent"`    // CPU使用百分比
	MemoryPercent float64 `json:"memoryPercent"` // 内存使用百分比
	DiskPercent   float64 `json:"diskPercent"`   // 磁盘使用百分比
	NetworkIn     int64   `json:"networkIn"`     // 网络入流量
	NetworkOut    int64   `json:"networkOut"`    // 网络出流量
}

// EventCallback 事件回调函数
type EventCallback func(event *SchedulerEvent)

// SchedulerOptions 调度器选项
type SchedulerOptions struct {
	// 基本配置
	NodeID              string              `json:"nodeId"`              // 节点ID
	EtcdKeyPrefix       string              `json:"etcdKeyPrefix"`       // ETCD键前缀
	HeartbeatInterval   time.Duration       `json:"heartbeatInterval"`   // 心跳间隔
	TaskTimeout         time.Duration       `json:"taskTimeout"`         // 任务超时时间
	WorkerTimeout       time.Duration       `json:"workerTimeout"`       // 工作节点超时时间
	WorkerTTL           time.Duration       `json:"workerTTL"`           // 工作节点租约TTL时间
	ElectionTimeout     time.Duration       `json:"electionTimeout"`     // 选举超时时间
	LoadBalanceStrategy LoadBalanceStrategy `json:"loadBalanceStrategy"` // 负载均衡策略

	// 高级配置
	EnableMetrics    bool          `json:"enableMetrics"`    // 启用指标收集
	EnableTracing    bool          `json:"enableTracing"`    // 启用链路追踪
	MaxRetryAttempts int           `json:"maxRetryAttempts"` // 最大重试次数
	RetryBackoffBase time.Duration `json:"retryBackoffBase"` // 重试退避基数
	RetryBackoffMax  time.Duration `json:"retryBackoffMax"`  // 重试退避最大值
	TaskQueueSize    int           `json:"taskQueueSize"`    // 任务队列大小
	WorkerPoolSize   int           `json:"workerPoolSize"`   // 工作池大小

	// 性能优化
	BatchSize           int           `json:"batchSize"`           // 批处理大小
	FlushInterval       time.Duration `json:"flushInterval"`       // 刷新间隔
	CompactionInterval  time.Duration `json:"compactionInterval"`  // 压缩间隔
	EnableTaskPipeline  bool          `json:"enableTaskPipeline"`  // 启用任务流水线
	EnableAsyncCallback bool          `json:"enableAsyncCallback"` // 启用异步回调
}

// DefaultSchedulerOptions 返回默认的调度器选项
func DefaultSchedulerOptions() *SchedulerOptions {
	return &SchedulerOptions{
		EtcdKeyPrefix:       "/scheduler/",
		HeartbeatInterval:   30 * time.Second,
		TaskTimeout:         5 * time.Minute,
		WorkerTimeout:       60 * time.Second,
		WorkerTTL:           90 * time.Second, // 工作节点租约TTL，设为心跳间隔的3倍
		ElectionTimeout:     30 * time.Second,
		LoadBalanceStrategy: StrategyLeastTasks,
		EnableMetrics:       true,
		EnableTracing:       false,
		MaxRetryAttempts:    3,
		RetryBackoffBase:    1 * time.Second,
		RetryBackoffMax:     60 * time.Second,
		TaskQueueSize:       10000,
		WorkerPoolSize:      100,
		BatchSize:           100,
		FlushInterval:       5 * time.Second,
		CompactionInterval:  1 * time.Hour,
		EnableTaskPipeline:  true,
		EnableAsyncCallback: true,
	}
}

// WorkerOptions 工作节点配置选项
type WorkerOptions struct {
	// 基本配置
	WorkerID          string        `json:"workerId"`          // 工作节点ID
	WorkerName        string        `json:"workerName"`        // 工作节点名称
	MaxConcurrency    int           `json:"maxConcurrency"`    // 最大并发数
	HeartbeatInterval time.Duration `json:"heartbeatInterval"` // 心跳间隔
	WorkerTTL         time.Duration `json:"workerTTL"`         // 工作节点TTL

	// 任务处理配置
	TaskTimeout       time.Duration `json:"taskTimeout"`       // 任务超时时间
	IdleTimeout       time.Duration `json:"idleTimeout"`       // 空闲超时时间
	EnableHealthCheck bool          `json:"enableHealthCheck"` // 启用健康检查

	// ETCD配置
	EtcdKeyPrefix string `json:"etcdKeyPrefix"` // ETCD键前缀

	// 性能配置
	BufferSize        int  `json:"bufferSize"`        // 缓冲区大小
	EnableCompression bool `json:"enableCompression"` // 启用压缩
	EnableMetrics     bool `json:"enableMetrics"`     // 启用指标收集
}

// DefaultWorkerOptions 返回默认的工作节点选项
func DefaultWorkerOptions() *WorkerOptions {
	return &WorkerOptions{
		MaxConcurrency:    10,
		HeartbeatInterval: 30 * time.Second,
		WorkerTTL:         90 * time.Second,
		TaskTimeout:       5 * time.Minute,
		IdleTimeout:       10 * time.Minute,
		EnableHealthCheck: true,
		EtcdKeyPrefix:     "/scheduler/",
		BufferSize:        1000,
		EnableCompression: false,
		EnableMetrics:     true,
	}
}
