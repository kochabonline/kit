package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestTask 测试任务结构
func TestTask(t *testing.T) {
	t.Run("创建任务", func(t *testing.T) {
		task := &Task{
			ID:       "test-task-1",
			Name:     "测试任务",
			Priority: TaskPriorityNormal,
			Status:   TaskStatusPending,
			Payload: map[string]any{
				"action": "test",
				"data":   "hello",
			},
			Metadata: map[string]string{
				"created_by": "test",
			},
			MaxRetries:    3,
			RetryInterval: 5 * time.Second,
			Timeout:       30 * time.Second,
			CreatedAt:     currentTimestamp(),
			UpdatedAt:     currentTimestamp(),
		}

		assert.Equal(t, "test-task-1", task.ID)
		assert.Equal(t, "测试任务", task.Name)
		assert.Equal(t, TaskPriorityNormal, task.Priority)
		assert.Equal(t, TaskStatusPending, task.Status)
		assert.Equal(t, "test", task.Payload["action"])
		assert.Equal(t, "test", task.Metadata["created_by"])
		assert.Equal(t, 3, task.MaxRetries)
		assert.True(t, task.CreatedAt > 0)
		assert.True(t, task.UpdatedAt > 0)
	})

	t.Run("任务状态转换", func(t *testing.T) {
		statuses := []TaskStatus{
			TaskStatusPending,
			TaskStatusScheduled,
			TaskStatusRunning,
			TaskStatusCompleted,
			TaskStatusFailed,
			TaskStatusCanceled,
			TaskStatusRetrying,
		}

		expectedStrings := []string{
			"pending",
			"scheduled",
			"running",
			"completed",
			"failed",
			"canceled",
			"retrying",
		}

		for i, status := range statuses {
			assert.Equal(t, expectedStrings[i], status.String())
		}
	})
}

// TestWorker 测试工作节点结构
func TestWorker(t *testing.T) {
	t.Run("创建工作节点", func(t *testing.T) {
		worker := &Worker{
			ID:             "worker-1",
			Name:           "测试工作节点",
			Status:         WorkerStatusOnline,
			Version:        "1.0.0",
			Capabilities:   []string{"compute", "storage"},
			MaxConcurrency: 10,
			Weight:         1,
			Tags: map[string]string{
				"env": "test",
			},
			LastHeartbeat: currentTimestamp(),
			RegisteredAt:  currentTimestamp(),
			UpdatedAt:     currentTimestamp(),
		}

		assert.Equal(t, "worker-1", worker.ID)
		assert.Equal(t, "测试工作节点", worker.Name)
		assert.Equal(t, WorkerStatusOnline, worker.Status)
		assert.Equal(t, "1.0.0", worker.Version)
		assert.Contains(t, worker.Capabilities, "compute")
		assert.Contains(t, worker.Capabilities, "storage")
		assert.Equal(t, 10, worker.MaxConcurrency)
		assert.Equal(t, "test", worker.Tags["env"])
		assert.True(t, worker.LastHeartbeat > 0)
	})

	t.Run("工作节点状态转换", func(t *testing.T) {
		statuses := []WorkerStatus{
			WorkerStatusOnline,
			WorkerStatusOffline,
			WorkerStatusBusy,
			WorkerStatusIdle,
		}

		expectedStrings := []string{
			"online",
			"offline",
			"busy",
			"idle",
		}

		for i, status := range statuses {
			assert.Equal(t, expectedStrings[i], status.String())
		}
	})
}

// TestTaskPriority 测试任务优先级
func TestTaskPriority(t *testing.T) {
	t.Run("优先级字符串转换", func(t *testing.T) {
		priorities := []TaskPriority{
			TaskPriorityLow,
			TaskPriorityNormal,
			TaskPriorityHigh,
			TaskPriorityCritical,
		}

		expectedStrings := []string{
			"low",
			"normal",
			"high",
			"critical",
		}

		for i, priority := range priorities {
			assert.Equal(t, expectedStrings[i], priority.String())
		}
	})
}

// TestEventType 测试事件类型
func TestEventType(t *testing.T) {
	t.Run("事件类型字符串转换", func(t *testing.T) {
		events := []EventType{
			EventTaskSubmitted,
			EventTaskScheduled,
			EventTaskStarted,
			EventTaskCompleted,
			EventTaskFailed,
			EventTaskCanceled,
			EventWorkerJoined,
			EventWorkerLeft,
			EventWorkerOnline,
			EventWorkerOffline,
			EventLeaderElected,
			EventLeaderLost,
			EventSchedulerStarted,
			EventSchedulerStopped,
		}

		expectedStrings := []string{
			"task_submitted",
			"task_scheduled",
			"task_started",
			"task_completed",
			"task_failed",
			"task_canceled",
			"worker_joined",
			"worker_left",
			"worker_online",
			"worker_offline",
			"leader_elected",
			"leader_lost",
			"scheduler_started",
			"scheduler_stopped",
		}

		for i, event := range events {
			assert.Equal(t, expectedStrings[i], event.String())
		}
	})
}

// TestSchedulerOptions 测试调度器选项
func TestSchedulerOptions(t *testing.T) {
	t.Run("默认选项", func(t *testing.T) {
		options := DefaultSchedulerOptions()

		assert.Equal(t, "/scheduler/", options.EtcdKeyPrefix)
		assert.Equal(t, 30*time.Second, options.HeartbeatInterval)
		assert.Equal(t, 5*time.Minute, options.TaskTimeout)
		assert.Equal(t, 60*time.Second, options.WorkerTimeout)
		assert.Equal(t, 90*time.Second, options.WorkerTTL)
		assert.Equal(t, 30*time.Second, options.ElectionTimeout)
		assert.Equal(t, StrategyLeastTasks, options.LoadBalanceStrategy)
		assert.True(t, options.EnableMetrics)
		assert.False(t, options.EnableTracing)
		assert.Equal(t, 3, options.MaxRetryAttempts)
		assert.Equal(t, 1*time.Second, options.RetryBackoffBase)
		assert.Equal(t, 60*time.Second, options.RetryBackoffMax)
		assert.Equal(t, 10000, options.TaskQueueSize)
		assert.Equal(t, 100, options.WorkerPoolSize)
		assert.Equal(t, 100, options.BatchSize)
		assert.Equal(t, 5*time.Second, options.FlushInterval)
		assert.Equal(t, 1*time.Hour, options.CompactionInterval)
		assert.True(t, options.EnableTaskPipeline)
		assert.True(t, options.EnableAsyncCallback)
	})

	t.Run("自定义选项", func(t *testing.T) {
		options := &SchedulerOptions{
			EtcdKeyPrefix:       "/custom-scheduler/",
			HeartbeatInterval:   15 * time.Second,
			TaskTimeout:         10 * time.Minute,
			WorkerTimeout:       120 * time.Second,
			ElectionTimeout:     60 * time.Second,
			LoadBalanceStrategy: StrategyRoundRobin,
			EnableMetrics:       false,
			EnableTracing:       true,
			MaxRetryAttempts:    5,
			BatchSize:           50,
		}

		assert.Equal(t, "/custom-scheduler/", options.EtcdKeyPrefix)
		assert.Equal(t, 15*time.Second, options.HeartbeatInterval)
		assert.Equal(t, 10*time.Minute, options.TaskTimeout)
		assert.Equal(t, 120*time.Second, options.WorkerTimeout)
		assert.Equal(t, 60*time.Second, options.ElectionTimeout)
		assert.Equal(t, StrategyRoundRobin, options.LoadBalanceStrategy)
		assert.False(t, options.EnableMetrics)
		assert.True(t, options.EnableTracing)
		assert.Equal(t, 5, options.MaxRetryAttempts)
		assert.Equal(t, 50, options.BatchSize)
	})
}

// TestWorkerOptions 测试工作节点选项
func TestWorkerOptions(t *testing.T) {
	t.Run("默认工作节点选项", func(t *testing.T) {
		options := DefaultWorkerOptions()

		assert.Equal(t, 10, options.MaxConcurrency)
		assert.Equal(t, 30*time.Second, options.HeartbeatInterval)
		assert.Equal(t, 90*time.Second, options.WorkerTTL)
		assert.Equal(t, 5*time.Minute, options.TaskTimeout)
		assert.Equal(t, 10*time.Minute, options.IdleTimeout)
		assert.True(t, options.EnableHealthCheck)
		assert.Equal(t, "/scheduler/", options.EtcdKeyPrefix)
		assert.Equal(t, 1000, options.BufferSize)
		assert.False(t, options.EnableCompression)
		assert.True(t, options.EnableMetrics)
	})
}

// TestTaskFilter 测试任务过滤器
func TestTaskFilter(t *testing.T) {
	t.Run("创建任务过滤器", func(t *testing.T) {
		filter := &TaskFilter{
			Status:        []TaskStatus{TaskStatusPending, TaskStatusRunning},
			WorkerID:      "worker-1",
			Priority:      []TaskPriority{TaskPriorityHigh, TaskPriorityCritical},
			CreatedAfter:  currentTimestamp() - 3600000, // 1小时前
			CreatedBefore: currentTimestamp(),
			Limit:         100,
			Offset:        0,
		}

		assert.Contains(t, filter.Status, TaskStatusPending)
		assert.Contains(t, filter.Status, TaskStatusRunning)
		assert.Equal(t, "worker-1", filter.WorkerID)
		assert.Contains(t, filter.Priority, TaskPriorityHigh)
		assert.Contains(t, filter.Priority, TaskPriorityCritical)
		assert.True(t, filter.CreatedAfter > 0)
		assert.True(t, filter.CreatedBefore > 0)
		assert.Equal(t, 100, filter.Limit)
		assert.Equal(t, 0, filter.Offset)
	})
}

// TestWorkerFilter 测试工作节点过滤器
func TestWorkerFilter(t *testing.T) {
	t.Run("创建工作节点过滤器", func(t *testing.T) {
		filter := &WorkerFilter{
			Status:       []WorkerStatus{WorkerStatusOnline, WorkerStatusIdle},
			Capabilities: []string{"compute", "storage"},
			Tags: map[string]string{
				"env": "production",
			},
			Limit:  50,
			Offset: 0,
		}

		assert.Contains(t, filter.Status, WorkerStatusOnline)
		assert.Contains(t, filter.Status, WorkerStatusIdle)
		assert.Contains(t, filter.Capabilities, "compute")
		assert.Contains(t, filter.Capabilities, "storage")
		assert.Equal(t, "production", filter.Tags["env"])
		assert.Equal(t, 50, filter.Limit)
		assert.Equal(t, 0, filter.Offset)
	})
}

// TestSchedulerEvent 测试调度器事件
func TestSchedulerEvent(t *testing.T) {
	t.Run("创建调度器事件", func(t *testing.T) {
		event := &SchedulerEvent{
			Type:      EventTaskSubmitted,
			TaskID:    "task-1",
			WorkerID:  "worker-1",
			Timestamp: currentTimestamp(),
			Data: map[string]any{
				"priority": "high",
				"category": "compute",
			},
		}

		assert.Equal(t, EventTaskSubmitted, event.Type)
		assert.Equal(t, "task-1", event.TaskID)
		assert.Equal(t, "worker-1", event.WorkerID)
		assert.True(t, event.Timestamp > 0)
		assert.Equal(t, "high", event.Data["priority"])
		assert.Equal(t, "compute", event.Data["category"])
	})
}

// TestSchedulerMetrics 测试调度器指标
func TestSchedulerMetrics(t *testing.T) {
	t.Run("创建调度器指标", func(t *testing.T) {
		metrics := &SchedulerMetrics{
			TasksTotal:         100,
			TasksPending:       10,
			TasksRunning:       20,
			TasksCompleted:     60,
			TasksFailed:        8,
			TasksCanceled:      2,
			TasksRetrying:      0,
			WorkersTotal:       5,
			WorkersOnline:      4,
			WorkersOffline:     1,
			WorkersBusy:        2,
			WorkersIdle:        2,
			AvgTaskDuration:    5 * time.Second,
			TaskThroughput:     10.5,
			SchedulerUptime:    2 * time.Hour,
			LastScheduleTime:   currentTimestamp(),
			TotalScheduleCount: 150,
			MemoryUsage:        1024 * 1024 * 100, // 100MB
			GoroutineCount:     50,
			UpdatedAt:          currentTimestamp(),
		}

		assert.Equal(t, int64(100), metrics.TasksTotal)
		assert.Equal(t, int64(10), metrics.TasksPending)
		assert.Equal(t, int64(20), metrics.TasksRunning)
		assert.Equal(t, int64(60), metrics.TasksCompleted)
		assert.Equal(t, int64(8), metrics.TasksFailed)
		assert.Equal(t, int64(2), metrics.TasksCanceled)
		assert.Equal(t, int64(0), metrics.TasksRetrying)
		assert.Equal(t, int64(5), metrics.WorkersTotal)
		assert.Equal(t, int64(4), metrics.WorkersOnline)
		assert.Equal(t, int64(1), metrics.WorkersOffline)
		assert.Equal(t, int64(2), metrics.WorkersBusy)
		assert.Equal(t, int64(2), metrics.WorkersIdle)
		assert.Equal(t, 5*time.Second, metrics.AvgTaskDuration)
		assert.Equal(t, 10.5, metrics.TaskThroughput)
		assert.Equal(t, 2*time.Hour, metrics.SchedulerUptime)
		assert.True(t, metrics.LastScheduleTime > 0)
		assert.Equal(t, int64(150), metrics.TotalScheduleCount)
		assert.Equal(t, int64(1024*1024*100), metrics.MemoryUsage)
		assert.Equal(t, 50, metrics.GoroutineCount)
		assert.True(t, metrics.UpdatedAt > 0)
	})
}

// TestHealthStatus 测试健康状态
func TestHealthStatus(t *testing.T) {
	t.Run("创建健康状态", func(t *testing.T) {
		health := &HealthStatus{
			CPU:    0.25,
			Memory: 0.60,
			Disk:   0.40,
			Load:   1.5,
		}

		assert.Equal(t, 0.25, health.CPU)
		assert.Equal(t, 0.60, health.Memory)
		assert.Equal(t, 0.40, health.Disk)
		assert.Equal(t, 1.5, health.Load)
	})
}

// TestResourceUsage 测试资源使用情况
func TestResourceUsage(t *testing.T) {
	t.Run("创建资源使用情况", func(t *testing.T) {
		usage := &ResourceUsage{
			CPUPercent:    25.5,
			MemoryPercent: 60.0,
			DiskPercent:   40.0,
			NetworkIn:     1024 * 1024,     // 1MB
			NetworkOut:    2 * 1024 * 1024, // 2MB
		}

		assert.Equal(t, 25.5, usage.CPUPercent)
		assert.Equal(t, 60.0, usage.MemoryPercent)
		assert.Equal(t, 40.0, usage.DiskPercent)
		assert.Equal(t, int64(1024*1024), usage.NetworkIn)
		assert.Equal(t, int64(2*1024*1024), usage.NetworkOut)
	})
}

// TestTaskExecutionResult 测试任务执行结果
func TestTaskExecutionResult(t *testing.T) {
	t.Run("创建任务执行结果", func(t *testing.T) {
		startTime := currentTimestamp()
		endTime := startTime + 5000 // 5秒后

		result := &TaskExecutionResult{
			TaskID:      "task-1",
			Status:      TaskStatusCompleted,
			Result:      map[string]any{"output": "success", "count": 42},
			Error:       "",
			StartedAt:   startTime,
			CompletedAt: endTime,
			Duration:    5 * time.Second,
			Metrics:     map[string]any{"cpu_time": 3.2, "memory_peak": 256},
		}

		assert.Equal(t, "task-1", result.TaskID)
		assert.Equal(t, TaskStatusCompleted, result.Status)
		assert.Equal(t, "success", result.Result["output"])
		assert.Equal(t, 42, result.Result["count"])
		assert.Empty(t, result.Error)
		assert.Equal(t, startTime, result.StartedAt)
		assert.Equal(t, endTime, result.CompletedAt)
		assert.Equal(t, 5*time.Second, result.Duration)
		assert.Equal(t, 3.2, result.Metrics["cpu_time"])
		assert.Equal(t, 256, result.Metrics["memory_peak"])
	})

	t.Run("创建失败的任务执行结果", func(t *testing.T) {
		startTime := currentTimestamp()
		endTime := startTime + 2000 // 2秒后

		result := &TaskExecutionResult{
			TaskID:      "task-2",
			Status:      TaskStatusFailed,
			Result:      nil,
			Error:       "连接超时",
			StartedAt:   startTime,
			CompletedAt: endTime,
			Duration:    2 * time.Second,
			Metrics:     map[string]any{"attempts": 3, "last_error": "timeout"},
		}

		assert.Equal(t, "task-2", result.TaskID)
		assert.Equal(t, TaskStatusFailed, result.Status)
		assert.Nil(t, result.Result)
		assert.Equal(t, "连接超时", result.Error)
		assert.Equal(t, startTime, result.StartedAt)
		assert.Equal(t, endTime, result.CompletedAt)
		assert.Equal(t, 2*time.Second, result.Duration)
		assert.Equal(t, 3, result.Metrics["attempts"])
		assert.Equal(t, "timeout", result.Metrics["last_error"])
	})
}

// TestUtilityFunctions 测试工具函数
func TestUtilityFunctions(t *testing.T) {
	t.Run("生成ID函数", func(t *testing.T) {
		// 测试任务ID生成
		taskID1 := generateTaskID()
		taskID2 := generateTaskID()
		assert.NotEmpty(t, taskID1)
		assert.NotEmpty(t, taskID2)
		assert.NotEqual(t, taskID1, taskID2)
		assert.Contains(t, taskID1, "task-")

		// 测试工作节点ID生成
		workerID1 := generateWorkerID()
		workerID2 := generateWorkerID()
		assert.NotEmpty(t, workerID1)
		assert.NotEmpty(t, workerID2)
		assert.NotEqual(t, workerID1, workerID2)
		assert.Contains(t, workerID1, "worker-")

		// 测试节点ID生成
		nodeID1 := generateNodeID()
		nodeID2 := generateNodeID()
		assert.NotEmpty(t, nodeID1)
		assert.NotEmpty(t, nodeID2)
		assert.NotEqual(t, nodeID1, nodeID2)
		assert.Contains(t, nodeID1, "scheduler-")
	})

	t.Run("时间戳函数", func(t *testing.T) {
		timestamp1 := currentTimestamp()
		time.Sleep(1 * time.Millisecond)
		timestamp2 := currentTimestamp()

		assert.True(t, timestamp1 > 0)
		assert.True(t, timestamp2 > timestamp1)
	})
}

// TestErrorDefinitions 测试错误定义
func TestErrorDefinitions(t *testing.T) {
	t.Run("错误常量", func(t *testing.T) {
		errors := []error{
			ErrTaskNotFound,
			ErrWorkerNotFound,
			ErrTaskAlreadyExists,
			ErrWorkerAlreadyExists,
			ErrInvalidTaskStatus,
			ErrInvalidWorkerStatus,
			ErrSchedulerNotStarted,
			ErrSchedulerAlreadyStarted,
			ErrNoAvailableWorkers,
			ErrTaskTimeout,
			ErrTaskCanceled,
			ErrMaxRetriesExceeded,
			ErrWorkerOffline,
			ErrContextCanceled,
			ErrLeaderElectionFailed,
		}

		expectedMessages := []string{
			"task not found",
			"worker not found",
			"task already exists",
			"worker already exists",
			"invalid task status",
			"invalid worker status",
			"scheduler not started",
			"scheduler already started",
			"no available workers",
			"task timeout",
			"task canceled",
			"max retries exceeded",
			"worker offline",
			"context canceled",
			"leader election failed",
		}

		for i, err := range errors {
			assert.Error(t, err)
			assert.Equal(t, expectedMessages[i], err.Error())
		}
	})
}

// TestLoadBalanceStrategy 测试负载均衡策略
func TestLoadBalanceStrategy(t *testing.T) {
	t.Run("负载均衡策略枚举", func(t *testing.T) {
		strategies := []LoadBalanceStrategy{
			StrategyLeastTasks,
			StrategyRoundRobin,
			StrategyWeightedRoundRobin,
			StrategyRandom,
			StrategyConsistentHash,
			StrategyLeastConnections,
		}

		// 确保所有策略都是不同的值
		for i, strategy := range strategies {
			for j, otherStrategy := range strategies {
				if i != j {
					assert.NotEqual(t, strategy, otherStrategy)
				}
			}
		}
	})
}
