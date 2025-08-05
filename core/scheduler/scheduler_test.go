package scheduler

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	// "github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/store/etcd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// 初始化全局日志配置
// func init() {
// 	log.SetGlobalLogger(log.New(log.WithCaller()))
// }

// TestSchedulerIntegration 集成测试：从创建调度器到任务执行的完整流程
func TestSchedulerIntegration(t *testing.T) {
	// 跳过集成测试（需要运行中的ETCD）
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 创建ETCD客户端
	etcdClient, err := createTestETCDClient()
	require.NoError(t, err, "创建ETCD客户端失败")
	defer etcdClient.Close()

	// 清理测试数据
	defer cleanupTestData(ctx, etcdClient)

	t.Run("完整的调度器生命周期测试", func(t *testing.T) {
		// 1. 创建调度器
		scheduler, err := createTestScheduler(etcdClient)
		require.NoError(t, err, "创建调度器失败")

		// 2. 启动调度器
		err = scheduler.Start(ctx)
		require.NoError(t, err, "启动调度器失败")
		defer func() {
			err := scheduler.Stop(ctx)
			assert.NoError(t, err, "停止调度器失败")
		}()

		// 等待调度器完全启动
		time.Sleep(2 * time.Second)

		// 3. 创建并启动工作节点
		worker := createTestWorker()
		workerNode, err := createTestWorkerNode(etcdClient, worker.ID, worker.Name)
		require.NoError(t, err, "创建工作节点失败")

		err = workerNode.Start(ctx)
		require.NoError(t, err, "启动工作节点失败")
		defer workerNode.Stop(ctx)

		// 等待工作节点完全启动
		time.Sleep(1 * time.Second)

		// 4. 注册工作节点到调度器
		err = scheduler.RegisterWorker(ctx, worker)
		require.NoError(t, err, "注册工作节点失败")

		// 等待工作节点注册完成
		time.Sleep(1 * time.Second)

		// 5. 验证工作节点注册成功
		retrievedWorker, err := scheduler.GetWorker(ctx, worker.ID)
		require.NoError(t, err, "获取工作节点失败")
		assert.Equal(t, worker.ID, retrievedWorker.ID)
		assert.Equal(t, worker.Name, retrievedWorker.Name)

		// 6. 提交任务
		task := createTestTask()
		err = scheduler.SubmitTask(ctx, task)
		require.NoError(t, err, "提交任务失败")

		// 7. 验证任务提交成功
		retrievedTask, err := scheduler.GetTask(ctx, task.ID)
		require.NoError(t, err, "获取任务失败")
		assert.Equal(t, task.ID, retrievedTask.ID)
		assert.Equal(t, TaskStatusPending, retrievedTask.Status)

		// 8. 等待任务被调度并执行完成
		err = waitForTaskStatus(ctx, scheduler, task.ID, TaskStatusCompleted, 60*time.Second)
		require.NoError(t, err, "等待任务完成超时")

		// 9. 验证任务执行完成
		completedTask, err := scheduler.GetTask(ctx, task.ID)
		require.NoError(t, err, "获取完成后的任务失败")
		assert.Equal(t, TaskStatusCompleted, completedTask.Status)
		assert.Equal(t, worker.ID, completedTask.WorkerID)
		assert.True(t, completedTask.ScheduledAt > 0)
		assert.True(t, completedTask.StartedAt > 0)
		assert.True(t, completedTask.CompletedAt > 0)
		assert.True(t, completedTask.CompletedAt >= completedTask.StartedAt)

		// 10. 验证调度器指标
		metrics := scheduler.GetMetrics()
		assert.True(t, metrics.TasksTotal >= 1)
		assert.True(t, metrics.TasksCompleted >= 1)
		assert.True(t, metrics.WorkersTotal >= 1)
	})
}

// TestSchedulerMultipleTasksExecution 测试多个任务并发执行
func TestSchedulerMultipleTasksExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	etcdClient, err := createTestETCDClient()
	require.NoError(t, err)
	defer etcdClient.Close()
	defer cleanupTestData(ctx, etcdClient)

	t.Run("多任务并发执行测试", func(t *testing.T) {
		scheduler, err := createTestScheduler(etcdClient)
		require.NoError(t, err)

		err = scheduler.Start(ctx)
		require.NoError(t, err)
		defer scheduler.Stop(ctx)

		time.Sleep(2 * time.Second)

		// 注册多个工作节点
		workers := make([]*Worker, 3)
		workerNodes := make([]*WorkerNode, 3)
		for i := 0; i < 3; i++ {
			worker := createTestWorker()
			worker.ID = fmt.Sprintf("worker-%d", i+1)
			worker.Name = fmt.Sprintf("测试工作节点-%d", i+1)
			workers[i] = worker

			// 创建并启动工作节点实例
			workerNode, err := createTestWorkerNode(etcdClient, worker.ID, worker.Name)
			require.NoError(t, err)
			workerNodes[i] = workerNode

			err = workerNode.Start(ctx)
			require.NoError(t, err)
			defer workerNode.Stop(ctx)
		}

		// 等待工作节点完全启动
		time.Sleep(2 * time.Second)

		// 注册工作节点到调度器
		for _, worker := range workers {
			err = scheduler.RegisterWorker(ctx, worker)
			require.NoError(t, err)
		}

		time.Sleep(1 * time.Second)

		// 提交多个任务
		taskCount := 10000
		tasks := make([]*Task, taskCount)
		for i := 0; i < taskCount; i++ {
			task := createTestTask()
			task.ID = fmt.Sprintf("task-%d", i+1)
			task.Name = fmt.Sprintf("测试任务-%d", i+1)
			tasks[i] = task

			err = scheduler.SubmitTask(ctx, task)
			require.NoError(t, err)
		}

		// 等待所有任务被调度
		for _, task := range tasks {
			err = waitForTaskStatus(ctx, scheduler, task.ID, TaskStatusScheduled, 30*time.Second)
			require.NoError(t, err, "任务 %s 调度超时", task.ID)
		}

		// 等待所有任务被调度并执行完成
		completedCount := 0
		maxWaitTime := 120 * time.Second // 增加等待时间
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		timeout := time.After(maxWaitTime)

		for completedCount < taskCount {
			select {
			case <-timeout:
				t.Fatalf("等待任务完成超时，完成数量: %d/%d", completedCount, taskCount)
			case <-ticker.C:
				completedCount = 0
				for _, task := range tasks {
					finalTask, err := scheduler.GetTask(ctx, task.ID)
					if err == nil && finalTask.Status == TaskStatusCompleted {
						completedCount++
					}
				}
				t.Logf("已完成任务数量: %d/%d", completedCount, taskCount)
			}
		}

		assert.Equal(t, taskCount, completedCount, "完成的任务数量不匹配")

		// 验证指标
		metrics := scheduler.GetMetrics()
		assert.True(t, metrics.TasksTotal >= int64(taskCount))
		assert.True(t, metrics.TasksCompleted >= int64(taskCount))
		assert.True(t, metrics.WorkersTotal >= 3)
	})
}

// TestSchedulerFailoverScenario 测试故障转移场景
func TestSchedulerFailoverScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	etcdClient, err := createTestETCDClient()
	require.NoError(t, err)
	defer etcdClient.Close()
	defer cleanupTestData(ctx, etcdClient)

	t.Run("工作节点故障转移测试", func(t *testing.T) {
		scheduler, err := createTestScheduler(etcdClient)
		require.NoError(t, err)

		err = scheduler.Start(ctx)
		require.NoError(t, err)
		defer scheduler.Stop(ctx)

		time.Sleep(2 * time.Second)

		// 注册工作节点
		worker1 := createTestWorker()
		worker1.ID = "worker-1"
		worker1.Name = "工作节点-1"
		err = scheduler.RegisterWorker(ctx, worker1)
		require.NoError(t, err)

		worker2 := createTestWorker()
		worker2.ID = "worker-2"
		worker2.Name = "工作节点-2"
		err = scheduler.RegisterWorker(ctx, worker2)
		require.NoError(t, err)

		time.Sleep(1 * time.Second)

		// 提交任务
		task := createTestTask()
		err = scheduler.SubmitTask(ctx, task)
		require.NoError(t, err)

		// 等待任务被调度
		err = waitForTaskStatus(ctx, scheduler, task.ID, TaskStatusScheduled, 30*time.Second)
		require.NoError(t, err)

		// 获取被分配的任务
		scheduledTask, err := scheduler.GetTask(ctx, task.ID)
		require.NoError(t, err)
		assignedWorkerID := scheduledTask.WorkerID

		// 模拟工作节点故障（注销工作节点）
		err = scheduler.UnregisterWorker(ctx, assignedWorkerID)
		require.NoError(t, err)

		// 等待一段时间让调度器检测到工作节点故障
		time.Sleep(5 * time.Second)

		// 验证任务可以被重新调度到其他工作节点
		// （在实际场景中，需要实现任务重新调度逻辑）

		// 验证工作节点已被移除
		_, err = scheduler.GetWorker(ctx, assignedWorkerID)
		assert.Error(t, err, "故障工作节点应该已被移除")
	})
}

// TestSchedulerEventHandling 测试事件处理
func TestSchedulerEventHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	etcdClient, err := createTestETCDClient()
	require.NoError(t, err)
	defer etcdClient.Close()
	defer cleanupTestData(ctx, etcdClient)

	t.Run("事件处理测试", func(t *testing.T) {
		scheduler, err := createTestScheduler(etcdClient)
		require.NoError(t, err)

		// 设置事件计数器
		eventCounts := make(map[EventType]int)
		var eventMutex sync.Mutex

		// 注册事件回调
		eventTypes := []EventType{
			EventTaskSubmitted,
			EventTaskScheduled,
			EventTaskStarted,
			EventTaskCompleted,
			EventWorkerJoined,
			EventSchedulerStarted,
		}

		for _, eventType := range eventTypes {
			et := eventType // 捕获循环变量
			err = scheduler.RegisterEventCallback(et, func(event *SchedulerEvent) {
				eventMutex.Lock()
				eventCounts[et]++
				eventMutex.Unlock()
			})
			require.NoError(t, err)
		}

		err = scheduler.Start(ctx)
		require.NoError(t, err)
		defer scheduler.Stop(ctx)

		time.Sleep(2 * time.Second)

		// 创建并启动工作节点
		worker := createTestWorker()
		workerNode, err := createTestWorkerNode(etcdClient, worker.ID, worker.Name)
		require.NoError(t, err)

		err = workerNode.Start(ctx)
		require.NoError(t, err)
		defer workerNode.Stop(ctx)

		// 等待工作节点完全启动
		time.Sleep(1 * time.Second)

		// 注册工作节点到调度器
		err = scheduler.RegisterWorker(ctx, worker)
		require.NoError(t, err)

		// 提交任务
		task := createTestTask()
		err = scheduler.SubmitTask(ctx, task)
		require.NoError(t, err)

		// 等待任务被调度
		err = waitForTaskStatus(ctx, scheduler, task.ID, TaskStatusScheduled, 30*time.Second)
		require.NoError(t, err)

		// 等待任务被调度并执行完成
		err = waitForTaskStatus(ctx, scheduler, task.ID, TaskStatusCompleted, 60*time.Second)
		require.NoError(t, err)

		// 等待事件处理
		time.Sleep(2 * time.Second)

		// 验证事件计数
		eventMutex.Lock()
		defer eventMutex.Unlock()

		assert.True(t, eventCounts[EventSchedulerStarted] >= 1, "调度器启动事件应该被触发")
		assert.True(t, eventCounts[EventWorkerJoined] >= 1, "工作节点加入事件应该被触发")
		assert.True(t, eventCounts[EventTaskSubmitted] >= 1, "任务提交事件应该被触发")
		assert.True(t, eventCounts[EventTaskScheduled] >= 1, "任务调度事件应该被触发")
		assert.True(t, eventCounts[EventTaskStarted] >= 1, "任务开始事件应该被触发")
		assert.True(t, eventCounts[EventTaskCompleted] >= 1, "任务完成事件应该被触发")
	})
}

// 辅助函数

// createTestETCDClient 创建测试用的ETCD客户端
func createTestETCDClient() (*etcd.Etcd, error) {
	config := &etcd.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5,
		Username:    "root",
		Password:    "12345678",
	}

	return etcd.New(config)
}

// createTestScheduler 创建测试用的调度器
func createTestScheduler(etcdClient *etcd.Etcd) (Scheduler, error) {
	options := DefaultSchedulerOptions()
	options.EtcdKeyPrefix = "/test-scheduler/"
	options.HeartbeatInterval = 10 * time.Second
	options.TaskTimeout = 30 * time.Second
	options.WorkerTimeout = 30 * time.Second
	options.ElectionTimeout = 10 * time.Second
	options.BatchSize = 10
	// 使用轮询策略确保任务均匀分布到所有工作节点
	options.LoadBalanceStrategy = StrategyRoundRobin

	return NewScheduler(etcdClient, options)
}

// createTestWorker 创建测试用的工作节点
func createTestWorker() *Worker {
	return &Worker{
		ID:             generateWorkerID(),
		Name:           "测试工作节点",
		Status:         WorkerStatusOnline,
		Version:        "1.0.0",
		Capabilities:   []string{"test", "compute"},
		MaxConcurrency: 5,
		Weight:         1,
		Tags: map[string]string{
			"env":  "test",
			"type": "compute",
		},
		Metadata: map[string]string{
			"created_by": "test",
		},
		LastHeartbeat: currentTimestamp(),
		HealthStatus: HealthStatus{
			CPU:    0.1,
			Memory: 0.2,
			Disk:   0.1,
			Load:   0.5,
		},
		ResourceUsage: ResourceUsage{
			CPUPercent:    10.0,
			MemoryPercent: 20.0,
			DiskPercent:   10.0,
			NetworkIn:     1024,
			NetworkOut:    2048,
		},
	}
}

// createTestWorkerNode 创建测试用的工作节点实例
func createTestWorkerNode(etcdClient *etcd.Etcd, workerID, workerName string) (*WorkerNode, error) {
	options := DefaultWorkerOptions()
	options.WorkerID = workerID
	options.WorkerName = workerName
	options.EtcdKeyPrefix = "/test-scheduler/"
	options.HeartbeatInterval = 5 * time.Second
	options.WorkerTTL = 15 * time.Second
	options.TaskTimeout = 30 * time.Second
	options.MaxConcurrency = 5

	workerNode, err := NewWorkerNode(etcdClient, options)
	if err != nil {
		return nil, err
	}

	// 设置简单的任务处理器
	workerNode.SetTaskProcessor(&testTaskProcessor{})

	return workerNode, nil
}

// testTaskProcessor 测试任务处理器
type testTaskProcessor struct{}

func (p *testTaskProcessor) Process(ctx context.Context, task *Task) error {
	// 模拟任务处理
	time.Sleep(100 * time.Millisecond)
	return nil
}

// createTestTask 创建测试用的任务
func createTestTask() *Task {
	return &Task{
		ID:       generateTaskID(),
		Name:     "测试任务",
		Priority: TaskPriorityNormal,
		Status:   TaskStatusPending,
		Payload: map[string]any{
			"action": "test",
			"data":   "hello world",
			"count":  42,
		},
		Metadata: map[string]string{
			"created_by": "test",
			"category":   "compute",
		},
		MaxRetries:    3,
		RetryInterval: 5 * time.Second,
		Timeout:       30 * time.Second,
		Dependencies:  []string{},
	}
}

// waitForTaskStatus 等待任务达到指定状态
func waitForTaskStatus(ctx context.Context, scheduler Scheduler, taskID string, expectedStatus TaskStatus, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("等待任务状态超时: %s", expectedStatus.String())
		case <-ticker.C:
			task, err := scheduler.GetTask(ctx, taskID)
			if err != nil {
				continue
			}
			if task.Status == expectedStatus {
				return nil
			}
		}
	}
}

// cleanupTestData 清理测试数据
func cleanupTestData(ctx context.Context, etcdClient *etcd.Etcd) {
	prefix := "/test-scheduler/"
	_, err := etcdClient.Client.Delete(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		fmt.Printf("清理测试数据失败: %v\n", err)
	}
}

// TestSchedulerConcurrentOperations 测试并发操作
func TestSchedulerConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	etcdClient, err := createTestETCDClient()
	require.NoError(t, err)
	defer etcdClient.Close()
	defer cleanupTestData(ctx, etcdClient)

	t.Run("并发操作测试", func(t *testing.T) {
		scheduler, err := createTestScheduler(etcdClient)
		require.NoError(t, err)

		err = scheduler.Start(ctx)
		require.NoError(t, err)
		defer scheduler.Stop(ctx)

		time.Sleep(2 * time.Second)

		// 并发注册多个工作节点
		workerCount := 5
		var workerWG sync.WaitGroup
		workerErrors := make(chan error, workerCount)

		for i := 0; i < workerCount; i++ {
			workerWG.Add(1)
			go func(index int) {
				defer workerWG.Done()
				worker := createTestWorker()
				worker.ID = fmt.Sprintf("concurrent-worker-%d", index)
				worker.Name = fmt.Sprintf("并发工作节点-%d", index)

				if err := scheduler.RegisterWorker(ctx, worker); err != nil {
					workerErrors <- err
				}
			}(i)
		}

		workerWG.Wait()
		close(workerErrors)

		// 检查工作节点注册错误
		for err := range workerErrors {
			assert.NoError(t, err, "并发注册工作节点失败")
		}

		// 并发提交多个任务
		taskCount := 20
		var taskWG sync.WaitGroup
		taskErrors := make(chan error, taskCount)

		for i := 0; i < taskCount; i++ {
			taskWG.Add(1)
			go func(index int) {
				defer taskWG.Done()
				task := createTestTask()
				task.ID = fmt.Sprintf("concurrent-task-%d", index)
				task.Name = fmt.Sprintf("并发任务-%d", index)

				if err := scheduler.SubmitTask(ctx, task); err != nil {
					taskErrors <- err
				}
			}(i)
		}

		taskWG.Wait()
		close(taskErrors)

		// 检查任务提交错误
		for err := range taskErrors {
			assert.NoError(t, err, "并发提交任务失败")
		}

		// 验证最终状态
		time.Sleep(5 * time.Second)

		workers, err := scheduler.ListWorkers(ctx, &WorkerFilter{})
		assert.NoError(t, err)
		assert.True(t, len(workers) >= workerCount, "注册的工作节点数量不够")

		tasks, err := scheduler.ListTasks(ctx, &TaskFilter{})
		assert.NoError(t, err)
		assert.True(t, len(tasks) >= taskCount, "提交的任务数量不够")

		metrics := scheduler.GetMetrics()
		assert.True(t, metrics.TasksTotal >= int64(taskCount))
		assert.True(t, metrics.WorkersTotal >= int64(workerCount))
	})
}

// BenchmarkSchedulerThroughput 调度器吞吐量基准测试
func BenchmarkSchedulerThroughput(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过基准测试")
	}

	ctx := context.Background()
	etcdClient, err := createTestETCDClient()
	require.NoError(b, err)
	defer etcdClient.Close()
	defer cleanupTestData(ctx, etcdClient)

	scheduler, err := createTestScheduler(etcdClient)
	require.NoError(b, err)

	err = scheduler.Start(ctx)
	require.NoError(b, err)
	defer scheduler.Stop(ctx)

	// 注册工作节点
	for i := 0; i < 10; i++ {
		worker := createTestWorker()
		worker.ID = fmt.Sprintf("bench-worker-%d", i)
		err = scheduler.RegisterWorker(ctx, worker)
		require.NoError(b, err)
	}

	time.Sleep(1 * time.Second)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			task := createTestTask()
			err := scheduler.SubmitTask(ctx, task)
			if err != nil {
				b.Error(err)
			}
		}
	})
}

// waitForAllTasksStatus 等待所有任务达到指定状态
func waitForAllTasksStatus(ctx context.Context, scheduler Scheduler, tasks []*Task, expectedStatus TaskStatus, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var wg sync.WaitGroup
	errChan := make(chan error, len(tasks))

	for _, task := range tasks {
		wg.Add(1)
		go func(t *Task) {
			defer wg.Done()
			if err := waitForTaskStatus(ctx, scheduler, t.ID, expectedStatus, timeout); err != nil {
				errChan <- fmt.Errorf("任务 %s 未能达到状态 %s: %w", t.ID, expectedStatus, err)
			}
		}(task)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		// 只返回第一个错误以避免日志泛滥
		return <-errChan
	}

	return nil
}

// BenchmarkSchedulerHighVolumeTasks 性能测试：高并发下的任务处理能力
func BenchmarkSchedulerHighVolumeTasks(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过性能测试")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	etcdClient, err := createTestETCDClient()
	require.NoError(b, err)
	defer etcdClient.Close()
	defer cleanupTestData(ctx, etcdClient)

	scheduler, err := createTestScheduler(etcdClient)
	require.NoError(b, err)

	err = scheduler.Start(ctx)
	require.NoError(b, err)
	defer scheduler.Stop(ctx)

	// 注册多个工作节点
	workerCount := 10
	workerNodes := make([]*WorkerNode, workerCount)
	for i := 0; i < workerCount; i++ {
		worker := createTestWorker()
		worker.ID = fmt.Sprintf("perf-worker-%d", i)
		worker.Name = fmt.Sprintf("性能测试工作节点-%d", i)

		workerNode, wErr := createTestWorkerNode(etcdClient, worker.ID, worker.Name)
		require.NoError(b, wErr)
		workerNodes[i] = workerNode

		wErr = workerNode.Start(ctx)
		require.NoError(b, wErr)
		defer workerNode.Stop(ctx)

		err = scheduler.RegisterWorker(ctx, worker)
		require.NoError(b, err)
	}

	// 等待所有组件启动
	time.Sleep(3 * time.Second)

	b.ResetTimer()

	// b.N是基准测试框架提供的迭代次数
	for i := 0; i < b.N; i++ {
		taskCount := 1000 // 每次迭代提交1000个任务
		tasks := make([]*Task, taskCount)
		var taskWG sync.WaitGroup

		// 停止计时器，准备提交任务
		b.StopTimer()
		for j := 0; j < taskCount; j++ {
			task := createTestTask()
			task.ID = fmt.Sprintf("perf-task-%d-%d", i, j)
			tasks[j] = task
		}
		// 重新开始计时器
		b.StartTimer()

		// 并发提交任务
		for _, task := range tasks {
			taskWG.Add(1)
			go func(t *Task) {
				defer taskWG.Done()
				if err := scheduler.SubmitTask(ctx, t); err != nil {
					b.Errorf("提交任务失败: %v", err)
				}
			}(task)
		}
		taskWG.Wait()

		// 等待所有任务完成
		err = waitForAllTasksStatus(ctx, scheduler, tasks, TaskStatusCompleted, 5*time.Minute)
		if err != nil {
			b.Fatalf("等待任务完成超时: %v", err)
		}
	}
}
