# Redis 分布式任务调度器

基于Redis实现的分布式任务调度器，提供高可用、高性能的任务调度服务。参考了etcd版本的设计，但使用Redis作为底层存储，提供更好的性能和简化的部署。

## 特性

- **分布式协调**: 基于Redis实现分布式锁和Leader选举
- **高可用性**: 支持多实例部署，自动故障切换
- **任务优先级**: 支持任务优先级队列和延时调度
- **工作节点管理**: 自动发现和健康检查工作节点
- **事件驱动**: 基于Redis Pub/Sub和Streams的事件系统
- **负载均衡**: 多种负载均衡策略（轮询、最少任务、一致性哈希等）
- **监控告警**: 全面的性能指标收集和告警机制

## 架构组件

### 核心组件

1. **Scheduler**: 主调度器，协调所有组件工作
2. **TaskManager**: 任务生命周期管理
3. **WorkerManager**: 工作节点注册和发现
4. **EventManager**: 事件发布订阅系统
5. **LoadBalancer**: 负载均衡器
6. **DistributedLock**: 分布式锁管理
7. **LeaderElector**: Leader选举器
8. **MonitoringSystem**: 监控和告警系统

### Redis数据结构

- **Hash**: 存储任务和工作节点详细信息
- **List**: 实现FIFO任务队列
- **ZSet**: 实现优先级队列和延时队列
- **Set**: 跟踪在线工作节点
- **Pub/Sub**: 实时事件通知
- **Streams**: 可靠的事件日志

## 快速开始

### 安装依赖

```bash
go get github.com/redis/go-redis/v9
```

### 基本使用

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/redis/go-redis/v9"
    "your-module/core/scheduler/redis"
)

func main() {
    // 创建Redis客户端
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    // 创建调度器配置
    config := &scheduler.Config{
        RedisClient:      rdb,
        SchedulerID:      "scheduler-1",
        CleanupInterval:  time.Minute * 5,
        TaskRetryLimit:   3,
        WorkerTimeout:    time.Minute * 2,
        LeaderLeaseTTL:   time.Second * 30,
        EnableMonitoring: true,
    }

    // 创建调度器实例
    sched, err := scheduler.NewRedisScheduler(config)
    if err != nil {
        log.Fatal("Failed to create scheduler:", err)
    }

    // 启动调度器
    ctx := context.Background()
    if err := sched.Start(ctx); err != nil {
        log.Fatal("Failed to start scheduler:", err)
    }
    defer sched.Stop()

    // 提交任务
    task := &scheduler.Task{
        ID:          "task-001",
        Type:        "data_processing",
        Payload:     `{"input": "data.csv", "output": "result.json"}`,
        Priority:    5,
        MaxRetries:  3,
        Timeout:     time.Minute * 10,
        ScheduleAt:  time.Now(),
    }

    if err := sched.SubmitTask(ctx, task); err != nil {
        log.Printf("Failed to submit task: %v", err)
    }

    // 注册工作节点
    worker := &scheduler.Worker{
        ID:       "worker-001",
        Address:  "http://worker-001:8080",
        TaskTypes: []string{"data_processing", "image_resize"},
        Capacity: 10,
    }

    if err := sched.RegisterWorker(ctx, worker); err != nil {
        log.Printf("Failed to register worker: %v", err)
    }

    // 保持运行
    select {}
}
```

### 工作节点实现

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "time"

    "your-module/core/scheduler/redis"
)

type WorkerServer struct {
    scheduler scheduler.Scheduler
    workerID  string
}

func (w *WorkerServer) handleTask(rw http.ResponseWriter, req *http.Request) {
    var task scheduler.Task
    if err := json.NewDecoder(req.Body).Decode(&task); err != nil {
        http.Error(rw, err.Error(), http.StatusBadRequest)
        return
    }

    // 处理任务逻辑
    log.Printf("Processing task: %s", task.ID)
    
    // 模拟任务处理
    time.Sleep(time.Second * 2)
    
    // 更新任务状态
    ctx := context.Background()
    if err := w.scheduler.UpdateTaskStatus(ctx, task.ID, scheduler.TaskStatusCompleted, "Task completed successfully"); err != nil {
        log.Printf("Failed to update task status: %v", err)
        http.Error(rw, err.Error(), http.StatusInternalServerError)
        return
    }

    rw.WriteHeader(http.StatusOK)
    json.NewEncoder(rw).Encode(map[string]string{"status": "completed"})
}

func main() {
    // ... 初始化调度器和注册工作节点

    server := &WorkerServer{
        scheduler: sched,
        workerID:  "worker-001",
    }

    http.HandleFunc("/execute", server.handleTask)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## 配置说明

### 调度器配置

```go
type Config struct {
    RedisClient      redis.UniversalClient // Redis客户端
    SchedulerID      string                // 调度器唯一标识
    Namespace        string                // Redis键命名空间，默认"scheduler"
    CleanupInterval  time.Duration         // 清理任务间隔，默认5分钟
    TaskRetryLimit   int                   // 任务重试次数，默认3
    WorkerTimeout    time.Duration         // 工作节点超时时间，默认2分钟
    LeaderLeaseTTL   time.Duration         // Leader租约TTL，默认30秒
    EnableMonitoring bool                  // 是否启用监控，默认false
    LoadBalanceStrategy string            // 负载均衡策略，默认"least_tasks"
}
```

### 负载均衡策略

- `least_tasks`: 选择任务数最少的工作节点
- `round_robin`: 轮询选择工作节点  
- `random`: 随机选择工作节点
- `weighted_round_robin`: 加权轮询
- `consistent_hash`: 一致性哈希
- `capacity_based`: 基于容量的选择

### 监控配置

```go
type MonitoringConfig struct {
    MetricsInterval   time.Duration // 指标收集间隔，默认30秒
    HealthCheckInterval time.Duration // 健康检查间隔，默认1分钟
    AlertRules        []AlertRule   // 告警规则
}

type AlertRule struct {
    Name      string        // 规则名称
    Metric    string        // 监控指标
    Threshold float64       // 阈值
    Operator  string        // 操作符: >, <, >=, <=, ==
    Duration  time.Duration // 持续时间
}
```

## API 参考

### 调度器接口

```go
type Scheduler interface {
    // 启动调度器
    Start(ctx context.Context) error
    
    // 停止调度器
    Stop() error
    
    // 提交任务
    SubmitTask(ctx context.Context, task *Task) error
    
    // 取消任务
    CancelTask(ctx context.Context, taskID string) error
    
    // 更新任务状态
    UpdateTaskStatus(ctx context.Context, taskID string, status TaskStatus, message string) error
    
    // 获取任务信息
    GetTask(ctx context.Context, taskID string) (*Task, error)
    
    // 列出任务
    ListTasks(ctx context.Context, status TaskStatus, limit int) ([]*Task, error)
    
    // 注册工作节点
    RegisterWorker(ctx context.Context, worker *Worker) error
    
    // 注销工作节点
    UnregisterWorker(ctx context.Context, workerID string) error
    
    // 获取工作节点信息
    GetWorker(ctx context.Context, workerID string) (*Worker, error)
    
    // 列出工作节点
    ListWorkers(ctx context.Context) ([]*Worker, error)
    
    // 获取调度器状态
    GetStatus(ctx context.Context) (*SchedulerStatus, error)
}
```

### 任务状态

```go
type TaskStatus string

const (
    TaskStatusPending    TaskStatus = "pending"    // 等待中
    TaskStatusRunning    TaskStatus = "running"    // 执行中
    TaskStatusCompleted  TaskStatus = "completed"  // 已完成
    TaskStatusFailed     TaskStatus = "failed"     // 失败
    TaskStatusCancelled  TaskStatus = "cancelled"  // 已取消
    TaskStatusRetrying   TaskStatus = "retrying"   // 重试中
)
```

### 工作节点状态

```go
type WorkerStatus string

const (
    WorkerStatusOnline   WorkerStatus = "online"   // 在线
    WorkerStatusOffline  WorkerStatus = "offline"  // 离线
    WorkerStatusBusy     WorkerStatus = "busy"     // 繁忙
)
```

## 监控和告警

### 内置指标

- `tasks_submitted_total`: 提交的任务总数
- `tasks_completed_total`: 完成的任务总数  
- `tasks_failed_total`: 失败的任务总数
- `tasks_pending`: 等待中的任务数
- `tasks_running`: 执行中的任务数
- `workers_online`: 在线工作节点数
- `scheduler_is_leader`: 是否为Leader (1/0)
- `task_execution_duration_seconds`: 任务执行时长
- `worker_utilization`: 工作节点利用率

### 告警规则示例

```go
alertRules := []scheduler.AlertRule{
    {
        Name:      "high_task_failure_rate",
        Metric:    "task_failure_rate",
        Threshold: 0.1, // 失败率超过10%
        Operator:  ">",
        Duration:  time.Minute * 5,
    },
    {
        Name:      "no_online_workers",
        Metric:    "workers_online",
        Threshold: 1,
        Operator:  "<",
        Duration:  time.Minute,
    },
}
```

## 最佳实践

### 性能优化

1. **Redis连接池**: 配置合适的连接池大小
2. **批量操作**: 使用Pipeline进行批量Redis操作
3. **键过期**: 设置合理的键过期时间避免内存泄漏
4. **监控指标**: 定期清理过期的监控数据

### 高可用部署

1. **Redis集群**: 使用Redis Cluster或Sentinel实现高可用
2. **多调度器实例**: 部署多个调度器实例，通过Leader选举保证一致性
3. **工作节点冗余**: 部署足够的工作节点应对故障
4. **监控告警**: 配置完善的监控告警体系

### 安全考虑

1. **Redis认证**: 启用Redis密码认证
2. **网络隔离**: 使用VPC或防火墙限制网络访问
3. **TLS加密**: 启用Redis TLS加密传输
4. **权限控制**: 使用Redis ACL限制操作权限

## 故障排查

### 常见问题

1. **Leader选举失败**
   - 检查Redis连接是否正常
   - 确认时钟同步
   - 查看Leader租约TTL设置

2. **任务调度延迟**
   - 检查工作节点是否在线
   - 查看任务队列积压情况
   - 确认负载均衡策略是否合适

3. **工作节点离线**
   - 检查心跳机制
   - 确认网络连接
   - 查看工作节点健康状态

### 日志和调试

启用详细日志记录：

```go
// 在创建调度器时启用调试模式
config.Debug = true
config.LogLevel = "debug"
```

查看Redis键空间：

```bash
# 查看所有调度器相关键
redis-cli --scan --pattern "scheduler:*"

# 查看任务队列
redis-cli LLEN scheduler:tasks:pending

# 查看在线工作节点
redis-cli SMEMBERS scheduler:workers:online
```