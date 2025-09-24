# Scheduler - 事件驱动分布式任务调度器

一个基于 ETCD 的高性能、事件驱动的分布式任务调度系统，支持多种负载均衡策略、自动故障恢复和实时监控。

## 🚀 特性

### 核心功能
- **事件驱动架构**：完全基于事件的异步调度，高性能低延迟
- **分布式调度**：支持多节点部署，自动 Leader 选举
- **多种负载均衡策略**：最少任务数、轮询、加权轮询、随机、一致性哈希等
- **故障自动恢复**：工作节点下线时自动重新调度任务
- **实时监控**：丰富的指标收集和事件回调机制

### 高级特性
- **任务优先级**：支持低、普通、高、紧急四种优先级
- **任务重试**：可配置的重试策略和退避算法
- **健康检查**：工作节点健康状态监控
- **资源监控**：CPU、内存、网络等资源使用情况
- **动态扩缩容**：工作节点可动态加入和离开

## 📋 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Scheduler A   │    │   Scheduler B   │    │   Scheduler C   │
│   (Leader)      │    │   (Follower)    │    │   (Follower)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                        ┌─────────────────┐
                        │      ETCD       │
                        │   (分布式存储)   │
                        └─────────────────┘
                                 │
         ┌───────────────────────┼───────────────────────┐
         │                       │                       │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Worker Node   │    │   Worker Node   │    │   Worker Node   │
│       A         │    │       B         │    │       C         │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 🔧 快速开始

### 安装依赖

```bash
go mod tidy
```

### 基本使用

#### 1. 创建调度器

```go
package main

import (
    "context"
    "time"
    
    "github.com/kochabonline/kit/core/scheduler"
    "github.com/kochabonline/kit/store/etcd"
    "github.com/kochabonline/kit/log"
)

func main() {
    ctx := context.Background()
    
    // 创建 ETCD 客户端
    etcdClient, err := etcd.New(&etcd.Config{
        Endpoints: []string{"localhost:2379"},
    })
    if err != nil {
        log.Fatal().Err(err).Msg("failed to create etcd client")
    }
    defer etcdClient.Close()
    
    // 创建调度器选项
    options := &scheduler.SchedulerOptions{
        NodeID:              "scheduler-node-1",
        EtcdKeyPrefix:       "/my-app/scheduler/",
        HeartbeatInterval:   30 * time.Second,
        TaskTimeout:         5 * time.Minute,
        LoadBalanceStrategy: scheduler.StrategyLeastTasks,
        EnableMetrics:       true,
    }
    
    // 创建调度器
    sched, err := scheduler.NewScheduler(etcdClient, options)
    if err != nil {
        log.Fatal().Err(err).Msg("failed to create scheduler")
    }
    
    // 启动调度器
    if err := sched.Start(ctx); err != nil {
        log.Fatal().Err(err).Msg("failed to start scheduler")
    }
    defer sched.Stop(ctx)
    
    // 调度器现在已启动并准备接收任务
    log.Info().Msg("scheduler started successfully")
}
```

#### 2. 创建工作节点

```go
package main

import (
    "context"
    "time"
    
    "github.com/kochabonline/kit/core/scheduler"
    "github.com/kochabonline/kit/store/etcd"
    "github.com/kochabonline/kit/log"
)

// 实现任务处理器
type MyTaskProcessor struct{}

func (p *MyTaskProcessor) Process(ctx context.Context, task *scheduler.Task) error {
    log.Info().
        Str("taskId", task.ID).
        Str("taskName", task.Name).
        Msg("processing task")
    
    // 模拟任务处理
    time.Sleep(2 * time.Second)
    
    log.Info().
        Str("taskId", task.ID).
        Msg("task completed")
    
    return nil
}

func main() {
    ctx := context.Background()
    
    // 创建 ETCD 客户端
    etcdClient, err := etcd.New(&etcd.Config{
        Endpoints: []string{"localhost:2379"},
    })
    if err != nil {
        log.Fatal().Err(err).Msg("failed to create etcd client")
    }
    defer etcdClient.Close()
    
    // 创建工作节点选项
    options := &scheduler.WorkerOptions{
        WorkerID:          "worker-node-1",
        WorkerName:        "My Worker Node",
        MaxConcurrency:    5,
        HeartbeatInterval: 30 * time.Second,
        EtcdKeyPrefix:     "/my-app/scheduler/",
    }
    
    // 创建工作节点
    worker, err := scheduler.NewWorkerNode(etcdClient, options)
    if err != nil {
        log.Fatal().Err(err).Msg("failed to create worker")
    }
    
    // 设置任务处理器
    worker.SetTaskProcessor(&MyTaskProcessor{})
    
    // 启动工作节点
    if err := worker.Start(ctx); err != nil {
        log.Fatal().Err(err).Msg("failed to start worker")
    }
    defer worker.Stop(ctx)
    
    log.Info().Msg("worker started successfully")
    
    // 保持运行
    select {}
}
```

#### 3. 提交任务

```go
package main

import (
    "context"
    
    "github.com/kochabonline/kit/core/scheduler"
    "github.com/kochabonline/kit/store/etcd"
    "github.com/kochabonline/kit/log"
)

func main() {
    ctx := context.Background()
    
    // 创建 ETCD 客户端和调度器（省略...）
    
    // 创建任务
    task := &scheduler.Task{
        ID:       "task-001",
        Name:     "数据处理任务",
        Priority: scheduler.TaskPriorityHigh,
        Payload: map[string]any{
            "user_id": 12345,
            "action":  "process_data",
        },
        Metadata: map[string]string{
            "department": "analytics",
            "project":    "user-behavior",
        },
        MaxRetries:    3,
        RetryInterval: time.Minute,
        Timeout:       10 * time.Minute,
    }
    
    // 提交任务
    if err := sched.SubmitTask(ctx, task); err != nil {
        log.Error().Err(err).Msg("failed to submit task")
        return
    }
    
    log.Info().
        Str("taskId", task.ID).
        Msg("task submitted successfully")
    
    // 查询任务状态
    retrievedTask, err := sched.GetTask(ctx, task.ID)
    if err != nil {
        log.Error().Err(err).Msg("failed to get task")
        return
    }
    
    log.Info().
        Str("taskId", retrievedTask.ID).
        Str("status", retrievedTask.Status.String()).
        Str("workerId", retrievedTask.WorkerID).
        Msg("task status")
}
```

## 📊 负载均衡策略

### 支持的策略

1. **最少任务数** (`StrategyLeastTasks`)：选择当前任务数最少的工作节点
2. **轮询** (`StrategyRoundRobin`)：按顺序轮流分配任务
3. **加权轮询** (`StrategyWeightedRoundRobin`)：根据工作节点权重分配任务
4. **随机** (`StrategyRandom`)：随机选择工作节点
5. **一致性哈希** (`StrategyConsistentHash`)：基于任务ID的一致性哈希
6. **最少连接数** (`StrategyLeastConnections`)：选择连接数最少的工作节点

### 动态策略切换

```go
// 获取当前策略
currentStrategy := sched.GetLoadBalanceStrategy()
log.Info().Str("strategy", fmt.Sprintf("%v", currentStrategy)).Msg("current strategy")

// 切换策略
sched.SetLoadBalanceStrategy(scheduler.StrategyConsistentHash)

// 重置负载均衡器状态
sched.ResetLoadBalancer()
```

## 📈 监控和指标

### 事件回调

```go
// 注册事件回调
err := sched.RegisterEventCallback(scheduler.EventTaskCompleted, func(event *scheduler.SchedulerEvent) {
    log.Info().
        Str("taskId", event.TaskID).
        Str("workerId", event.WorkerID).
        Int64("timestamp", event.Timestamp).
        Msg("task completed event received")
})
if err != nil {
    log.Error().Err(err).Msg("failed to register event callback")
}

// 支持的事件类型
// - EventTaskSubmitted: 任务提交
// - EventTaskScheduled: 任务调度
// - EventTaskStarted: 任务开始
// - EventTaskCompleted: 任务完成
// - EventTaskFailed: 任务失败
// - EventTaskCanceled: 任务取消
// - EventWorkerJoined: 工作节点加入
// - EventWorkerLeft: 工作节点离开
// - EventWorkerOnline: 工作节点上线
// - EventWorkerOffline: 工作节点下线
```

### 指标收集

```go
// 获取调度器指标
metrics := sched.GetMetrics()

log.Info().
    Int64("tasksTotal", metrics.TasksTotal).
    Int64("tasksPending", metrics.TasksPending).
    Int64("tasksRunning", metrics.TasksRunning).
    Int64("tasksCompleted", metrics.TasksCompleted).
    Int64("tasksFailed", metrics.TasksFailed).
    Int64("workersOnline", metrics.WorkersOnline).
    Int64("workersOffline", metrics.WorkersOffline).
    Float64("taskThroughput", metrics.TaskThroughput).
    Dur("schedulerUptime", metrics.SchedulerUptime).
    Msg("scheduler metrics")
```

## 🔄 任务生命周期

```
┌─────────────┐    SubmitTask     ┌─────────────┐
│   Created   │ ───────────────> │   Pending   │
└─────────────┘                  └─────────────┘
                                         │
                                  Schedule │
                                         ▼
┌─────────────┐                  ┌─────────────┐
│  Completed  │                  │  Scheduled  │
└─────────────┘                  └─────────────┘
        ▲                                │
        │                         Start │ 
        │                               ▼
┌─────────────┐                  ┌─────────────┐
│   Running   │ <─────────────── │   Running   │
└─────────────┘                  └─────────────┘
        │                                │
        │ Success                 Failure │
        ▼                                ▼
┌─────────────┐                  ┌─────────────┐
│  Completed  │                  │   Failed    │
└─────────────┘                  └─────────────┘
                                         │
                                  Retry │ (if enabled)
                                         ▼
                                 ┌─────────────┐
                                 │  Retrying   │
                                 └─────────────┘
```

## 🛡️ 高可用性

### Leader 选举

调度器支持多节点部署，自动进行 Leader 选举：

```go
// 检查当前节点是否为 Leader
if sched.IsLeader() {
    log.Info().Msg("current node is leader")
} else {
    log.Info().Msg("current node is follower")
}

// 获取节点ID
nodeID := sched.GetNodeID()
log.Info().Str("nodeId", nodeID).Msg("current node ID")
```

### 故障恢复

- 工作节点下线时，调度器自动检测并重新调度运行中的任务
- 调度器节点下线时，其他节点自动接管调度职责
- 支持工作节点动态加入和离开

## ⚙️ 配置说明

### 调度器配置

```go
options := &scheduler.SchedulerOptions{
    NodeID:              "scheduler-1",           // 节点ID
    EtcdKeyPrefix:       "/app/scheduler/",       // ETCD键前缀
    HeartbeatInterval:   30 * time.Second,        // 心跳间隔
    TaskTimeout:         5 * time.Minute,         // 任务超时时间
    WorkerTimeout:       60 * time.Second,        // 工作节点超时时间
    WorkerTTL:           90 * time.Second,        // 工作节点租约TTL
    ElectionTimeout:     30 * time.Second,        // 选举超时时间
    LoadBalanceStrategy: scheduler.StrategyLeastTasks, // 负载均衡策略
    EnableMetrics:       true,                    // 启用指标收集
    EnableTracing:       false,                   // 启用链路追踪
    MaxRetryAttempts:    3,                       // 最大重试次数
    RetryBackoffBase:    1 * time.Second,         // 重试退避基数
    RetryBackoffMax:     60 * time.Second,        // 重试退避最大值
    TaskQueueSize:       10000,                   // 任务队列大小
    WorkerPoolSize:      100,                     // 工作池大小
}
```

### 工作节点配置

```go
options := &scheduler.WorkerOptions{
    WorkerID:          "worker-1",                // 工作节点ID
    WorkerName:        "Data Processor",         // 工作节点名称
    MaxConcurrency:    10,                       // 最大并发数
    HeartbeatInterval: 30 * time.Second,         // 心跳间隔
    WorkerTTL:         90 * time.Second,         // 工作节点TTL
    TaskTimeout:       5 * time.Minute,          // 任务超时时间
    IdleTimeout:       10 * time.Minute,         // 空闲超时时间
    EnableHealthCheck: true,                     // 启用健康检查
    EtcdKeyPrefix:     "/app/scheduler/",        // ETCD键前缀
    BufferSize:        1000,                     // 缓冲区大小
    EnableCompression: false,                    // 启用压缩
    EnableMetrics:     true,                     // 启用指标收集
}
```

## 🔍 最佳实践

### 1. 任务设计

- 任务应该是幂等的，能够安全重试
- 将大任务拆分为多个小任务，提高并行度
- 合理设置任务超时时间和重试策略
- 使用任务优先级来确保重要任务优先处理

### 2. 工作节点配置

- 根据工作节点的性能调整 `MaxConcurrency`
- 设置合适的心跳间隔，平衡性能和故障检测速度
- 启用健康检查来监控工作节点状态

### 3. 监控和日志

- 注册关键事件的回调函数进行监控
- 定期收集和分析调度器指标
- 使用结构化日志记录关键操作

### 4. 错误处理

- 实现健壮的任务处理器，处理各种异常情况
- 合理设置重试次数和退避策略
- 监控失败任务并及时处理

## 📚 API 参考

### 主要接口

- `Scheduler`: 调度器主接口
- `TaskProcessor`: 任务处理器接口
- `LoadBalancer`: 负载均衡器接口

### 主要类型

- `Task`: 任务定义
- `Worker`: 工作节点定义
- `SchedulerEvent`: 调度器事件
- `SchedulerMetrics`: 调度器指标

详细的 API 文档请参考源代码中的接口定义和注释。
