package scheduler

import (
	"context"
	"testing"
	"time"

	storeRedis "github.com/kochabonline/kit/store/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRedisSchedulerBasicFlow 基础流程：启动 -> 成为Leader -> 注册Worker -> 提交任务 -> 任务被调度
func TestRedisSchedulerBasicFlow(t *testing.T) {
	if testing.Short() {
		// 短测试模式跳过
		t.Skip("跳过集成测试")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 创建 Redis 配置 (要求本地已启动 redis-server 无密码: 6379)
	cfg := &storeRedis.SingleConfig{Host: "localhost", Port: 6379, Password: "12345678", DB: 0}
	client, err := storeRedis.NewClient(cfg)
	if err != nil {
		t.Skipf("无法连接本地 Redis，跳过测试: %v", err)
	}
	defer client.Close()

	// 预清理可能残留的 leader 锁，避免上次测试异常中断导致无法成为 leader
	_ = client.Client.Del(ctx, "scheduler:lock:leader").Err()

	options := DefaultSchedulerOptions()
	options.NodeID = "test-node-1"
	// 使用>=1s，避免内部最小 TTL 限制产生噪声日志
	options.WorkerLeaseTTL = 1200 * time.Millisecond
	options.LeaderLeaseDuration = 3 * time.Second
	options.EnableMetrics = false   // 简化测试
	options.HealthCheckInterval = 0 // 关闭周期健康检查
	options.MetricsInterval = 0     // 关闭指标采集
	options.LoadBalanceStrategy = StrategyLeastTasks

	sched, err := NewScheduler(cfg, options)
	require.NoError(t, err, "创建调度器失败")

	// 启动调度器
	require.NoError(t, sched.Start(ctx), "启动调度器失败")
	defer sched.Stop(context.Background())

	// 等待成为 Leader
	ok := waitUntil(ctx, 5*time.Second, 100*time.Millisecond, func() bool { return sched.IsLeader() })
	if !ok {
		t.Fatalf("节点在超时时间内未成为 Leader")
	}

	// 注册事件计数回调（仅关心提交与调度事件）
	eventCounts := make(map[EventType]int)
	var eventTypes = []EventType{EventTaskSubmitted, EventTaskScheduled}
	for _, et := range eventTypes {
		etype := et
		err := sched.RegisterEventCallback(etype, func(c context.Context, e *SchedulerEvent) error {
			// 简单计数（不加锁：事件管理器回调本测试场景低并发，可接受；若需严格安全可加互斥）
			eventCounts[etype]++
			return nil
		})
		require.NoError(t, err)
	}

	// 注册一个 Worker
	worker := &Worker{
		ID:           "worker-1",
		Name:         "test-worker",
		Address:      "127.0.0.1",
		Port:         8080,
		Capabilities: []string{"test"},
		MaxTasks:     5,
		Weight:       1,
		Priority:     1,
	}
	require.NoError(t, sched.RegisterWorker(ctx, worker), "注册 Worker 失败")

	// 提交任务
	task := &Task{
		Name:     "demo-task",
		Priority: TaskPriorityNormal,
		Payload: map[string]any{
			"action": "echo",
			"value":  "hello",
		},
		Metadata:   map[string]string{"case": "basic"},
		MaxRetries: 1,
	}
	require.NoError(t, sched.SubmitTask(ctx, task), "提交任务失败")

	// 等待任务被调度 (状态变为 scheduled 并分配 worker)
	ok = waitUntil(ctx, 10*time.Second, 200*time.Millisecond, func() bool {
		got, err := sched.GetTask(ctx, task.ID)
		if err != nil || got == nil {
			return false
		}
		return got.Status == TaskStatusScheduled && got.WorkerID != ""
	})
	if !ok {
		t.Fatalf("任务在超时时间内未被调度: %s", task.ID)
	}
	time.Sleep(10 * time.Second)
	// 再次获取并断言
	finalTask, err := sched.GetTask(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, TaskStatusScheduled, finalTask.Status, "任务状态应为 scheduled")
	assert.Equal(t, worker.ID, finalTask.WorkerID, "任务应被分配给注册的 worker")
	assert.NotZero(t, finalTask.ScheduledAt, "任务应记录调度时间")

	// 健康检查
	require.NoError(t, sched.Health(ctx), "健康检查失败")

	// 事件断言（>=1 防止偶发延迟）
	assert.GreaterOrEqual(t, eventCounts[EventTaskSubmitted], 1, "应触发任务提交事件")
	assert.GreaterOrEqual(t, eventCounts[EventTaskScheduled], 1, "应触发任务调度事件")
}

// waitUntil 在超时时间内轮询条件函数，满足返回 true
func waitUntil(parent context.Context, timeout, interval time.Duration, cond func() bool) bool {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	for {
		if cond() {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(interval):
		}
	}
}
