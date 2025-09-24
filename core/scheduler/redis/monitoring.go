package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/kochabonline/kit/log"
	"github.com/redis/go-redis/v9"
)

// MetricsCollector 指标收集器
type MetricsCollector struct {
	client   redis.Cmdable
	keys     *RedisKeys
	nodeID   string
	interval time.Duration

	// 性能统计
	taskSubmissionRate float64 // 任务提交速率 (tasks/min)
	taskCompletionRate float64 // 任务完成速率 (tasks/min)
	avgSchedulingDelay int64   // 平均调度延迟 (ms)

	// 计数器
	lastSubmittedTasks int64     // 上次统计的已提交任务数
	lastCompletedTasks int64     // 上次统计的已完成任务数
	lastCollectTime    time.Time // 上次收集时间

	stopChan chan struct{}
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector(client redis.Cmdable, keys *RedisKeys, nodeID string, interval time.Duration) *MetricsCollector {
	return &MetricsCollector{
		client:          client,
		keys:            keys,
		nodeID:          nodeID,
		interval:        interval,
		lastCollectTime: time.Now(),
		stopChan:        make(chan struct{}),
	}
}

// Start 启动指标收集
func (mc *MetricsCollector) Start(ctx context.Context) {
	ticker := time.NewTicker(mc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-mc.stopChan:
			return
		case <-ticker.C:
			mc.collectAndSaveMetrics(ctx)
		}
	}
}

// Stop 停止指标收集
func (mc *MetricsCollector) Stop() {
	close(mc.stopChan)
}

// collectAndSaveMetrics 收集并保存指标
func (mc *MetricsCollector) collectAndSaveMetrics(ctx context.Context) {
	metrics := mc.collectSystemMetrics(ctx)

	// 保存指标到Redis
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal metrics")
		return
	}

	pipe := mc.client.Pipeline()

	// 保存当前指标
	pipe.HSet(ctx, mc.keys.MetricsHash, mc.nodeID, metricsJSON)

	// 保存指标历史 (使用时间戳作为Stream ID)
	streamValues := map[string]any{
		"nodeId":             mc.nodeID,
		"timestamp":          time.Now().Unix(),
		"pendingTasks":       metrics.PendingTasks,
		"runningTasks":       metrics.RunningTasks,
		"completedTasks":     metrics.CompletedTasks,
		"failedTasks":        metrics.FailedTasks,
		"onlineWorkers":      metrics.OnlineWorkers,
		"busyWorkers":        metrics.BusyWorkers,
		"isLeader":           metrics.IsLeader,
		"taskSubmissionRate": metrics.TaskSubmissionRate,
		"taskCompletionRate": metrics.TaskCompletionRate,
		"taskThroughput":     metrics.TaskThroughput,
	}

	pipe.XAdd(ctx, &redis.XAddArgs{
		Stream: mc.keys.MetricsHash + ":history",
		MaxLen: 1000, // 保留最近1000条记录
		Approx: true,
		Values: streamValues,
	})

	if _, err := pipe.Exec(ctx); err != nil {
		log.Error().Err(err).Msg("failed to save metrics")
	}
}

// collectSystemMetrics 收集系统指标
func (mc *MetricsCollector) collectSystemMetrics(ctx context.Context) *SystemMetrics {
	metrics := &SystemMetrics{
		NodeID:    mc.nodeID,
		Timestamp: time.Now().Unix(),
	}

	// 收集任务相关指标
	pipe := mc.client.Pipeline()

	// 待处理任务数
	pipe.ZCard(ctx, mc.keys.TaskPriorityQueue)
	pipe.LLen(ctx, mc.keys.TaskQueue)

	// 各状态任务数
	pipe.SCard(ctx, mc.keys.TaskRunning)
	pipe.SCard(ctx, mc.keys.TaskCompleted)
	pipe.SCard(ctx, mc.keys.TaskFailed)

	// 工作节点指标
	pipe.SCard(ctx, mc.keys.WorkerOnline)
	pipe.SCard(ctx, mc.keys.WorkerBusy)
	pipe.HLen(ctx, mc.keys.WorkerHash)

	results, err := pipe.Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to collect metrics")
		return metrics
	}

	// 解析结果
	if len(results) >= 7 {
		metrics.PendingTasks = results[0].(*redis.IntCmd).Val() + results[1].(*redis.IntCmd).Val()
		metrics.RunningTasks = results[2].(*redis.IntCmd).Val()
		metrics.CompletedTasks = results[3].(*redis.IntCmd).Val()
		metrics.FailedTasks = results[4].(*redis.IntCmd).Val()
		metrics.OnlineWorkers = results[5].(*redis.IntCmd).Val()
		metrics.BusyWorkers = results[6].(*redis.IntCmd).Val()
		metrics.TotalWorkers = results[7].(*redis.IntCmd).Val()
	}

	// 计算派生指标
	metrics.IdleWorkers = metrics.OnlineWorkers - metrics.BusyWorkers
	metrics.OfflineWorkers = metrics.TotalWorkers - metrics.OnlineWorkers - metrics.BusyWorkers

	if metrics.OnlineWorkers > 0 {
		metrics.WorkerUtilization = float64(metrics.BusyWorkers) / float64(metrics.OnlineWorkers) * 100
	}

	// 计算任务提交率和完成率
	mc.updateTaskRates(metrics)

	// 计算吞吐量
	metrics.TaskThroughput = mc.taskCompletionRate

	return metrics
}

// updateTaskRates 更新任务提交率和完成率
func (mc *MetricsCollector) updateTaskRates(metrics *SystemMetrics) {
	now := time.Now()
	timeDiff := now.Sub(mc.lastCollectTime).Minutes()

	if timeDiff > 0 {
		// 计算总任务数（粗略估算）
		totalSubmitted := metrics.CompletedTasks + metrics.FailedTasks + metrics.RunningTasks + metrics.PendingTasks
		totalCompleted := metrics.CompletedTasks + metrics.FailedTasks

		if mc.lastSubmittedTasks > 0 {
			submittedDiff := totalSubmitted - mc.lastSubmittedTasks
			mc.taskSubmissionRate = float64(submittedDiff) / timeDiff
		}

		if mc.lastCompletedTasks > 0 {
			completedDiff := totalCompleted - mc.lastCompletedTasks
			mc.taskCompletionRate = float64(completedDiff) / timeDiff
		}

		// 更新记录
		mc.lastSubmittedTasks = totalSubmitted
		mc.lastCompletedTasks = totalCompleted
		mc.lastCollectTime = now
	}

	// 更新metrics中的速率信息
	metrics.TaskSubmissionRate = mc.taskSubmissionRate
	metrics.TaskCompletionRate = mc.taskCompletionRate
}

// GetTaskSubmissionRate 获取任务提交速率
func (mc *MetricsCollector) GetTaskSubmissionRate() float64 {
	return mc.taskSubmissionRate
}

// GetTaskCompletionRate 获取任务完成速率
func (mc *MetricsCollector) GetTaskCompletionRate() float64 {
	return mc.taskCompletionRate
}

// GetAverageSchedulingDelay 获取平均调度延迟
func (mc *MetricsCollector) GetAverageSchedulingDelay() time.Duration {
	return time.Duration(mc.avgSchedulingDelay) * time.Millisecond
}

// UpdateSchedulingDelay 更新调度延迟
func (mc *MetricsCollector) UpdateSchedulingDelay(delay time.Duration) {
	// 使用简单的移动平均
	newDelay := delay.Milliseconds()
	if mc.avgSchedulingDelay == 0 {
		mc.avgSchedulingDelay = newDelay
	} else {
		// 使用加权移动平均，新值权重0.2
		mc.avgSchedulingDelay = int64(float64(mc.avgSchedulingDelay)*0.8 + float64(newDelay)*0.2)
	}
}

// SystemMetrics 系统指标
type SystemMetrics struct {
	NodeID    string `json:"nodeId"`
	Timestamp int64  `json:"timestamp"`
	IsLeader  bool   `json:"isLeader"`

	// 任务指标
	PendingTasks       int64   `json:"pendingTasks"`
	RunningTasks       int64   `json:"runningTasks"`
	CompletedTasks     int64   `json:"completedTasks"`
	FailedTasks        int64   `json:"failedTasks"`
	TaskThroughput     float64 `json:"taskThroughput"`     // 任务/分钟
	TaskSubmissionRate float64 `json:"taskSubmissionRate"` // 任务提交速率/分钟
	TaskCompletionRate float64 `json:"taskCompletionRate"` // 任务完成速率/分钟

	// 工作节点指标
	TotalWorkers      int64   `json:"totalWorkers"`
	OnlineWorkers     int64   `json:"onlineWorkers"`
	BusyWorkers       int64   `json:"busyWorkers"`
	IdleWorkers       int64   `json:"idleWorkers"`
	OfflineWorkers    int64   `json:"offlineWorkers"`
	WorkerUtilization float64 `json:"workerUtilization"` // 百分比

	// 性能指标
	AvgSchedulingDelay int64   `json:"avgSchedulingDelay"` // 毫秒
	ErrorRate          float64 `json:"errorRate"`          // 百分比

	// Redis指标
	RedisLatency     int64 `json:"redisLatency"` // 微秒
	RedisConnections int64 `json:"redisConnections"`
}

// HealthChecker 健康检查器
type HealthChecker struct {
	scheduler Scheduler
	interval  time.Duration
	stopChan  chan struct{}
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(scheduler Scheduler, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		scheduler: scheduler,
		interval:  interval,
		stopChan:  make(chan struct{}),
	}
}

// Start 启动健康检查
func (hc *HealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-hc.stopChan:
			return
		case <-ticker.C:
			hc.performHealthCheck(ctx)
		}
	}
}

// Stop 停止健康检查
func (hc *HealthChecker) Stop() {
	close(hc.stopChan)
}

// performHealthCheck 执行健康检查
func (hc *HealthChecker) performHealthCheck(ctx context.Context) {
	start := time.Now()

	if err := hc.scheduler.Health(ctx); err != nil {
		log.Error().Err(err).Msg("health check failed")
		return
	}

	duration := time.Since(start)
	log.Debug().Dur("duration", duration).Msg("health check passed")
}

// PerformanceMonitor 性能监控器
type PerformanceMonitor struct {
	client redis.Cmdable
	keys   *RedisKeys

	// 性能指标
	taskLatencies   []int64 // 任务延迟历史
	schedulingTimes []int64 // 调度时间历史
	maxHistorySize  int     // 最大历史记录数

	stopChan chan struct{}
}

// NewPerformanceMonitor 创建性能监控器
func NewPerformanceMonitor(client redis.Cmdable, keys *RedisKeys) *PerformanceMonitor {
	return &PerformanceMonitor{
		client:         client,
		keys:           keys,
		maxHistorySize: 1000,
		stopChan:       make(chan struct{}),
	}
}

// RecordTaskLatency 记录任务延迟
func (pm *PerformanceMonitor) RecordTaskLatency(latency time.Duration) {
	if len(pm.taskLatencies) >= pm.maxHistorySize {
		// 移除最旧的记录
		pm.taskLatencies = pm.taskLatencies[1:]
	}
	pm.taskLatencies = append(pm.taskLatencies, latency.Milliseconds())
}

// RecordSchedulingTime 记录调度时间
func (pm *PerformanceMonitor) RecordSchedulingTime(duration time.Duration) {
	if len(pm.schedulingTimes) >= pm.maxHistorySize {
		pm.schedulingTimes = pm.schedulingTimes[1:]
	}
	pm.schedulingTimes = append(pm.schedulingTimes, duration.Microseconds())
}

// GetAverageTaskLatency 获取平均任务延迟
func (pm *PerformanceMonitor) GetAverageTaskLatency() time.Duration {
	if len(pm.taskLatencies) == 0 {
		return 0
	}

	var total int64
	for _, latency := range pm.taskLatencies {
		total += latency
	}

	avg := total / int64(len(pm.taskLatencies))
	return time.Duration(avg) * time.Millisecond
}

// GetAverageSchedulingTime 获取平均调度时间
func (pm *PerformanceMonitor) GetAverageSchedulingTime() time.Duration {
	if len(pm.schedulingTimes) == 0 {
		return 0
	}

	var total int64
	for _, duration := range pm.schedulingTimes {
		total += duration
	}

	avg := total / int64(len(pm.schedulingTimes))
	return time.Duration(avg) * time.Microsecond
}

// AlertManager 告警管理器
type AlertManager struct {
	rules    []AlertRule
	notifier AlertNotifier
}

// AlertRule 告警规则
type AlertRule struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Condition   string        `json:"condition"` // 条件表达式
	Threshold   float64       `json:"threshold"` // 阈值
	Duration    time.Duration `json:"duration"`  // 持续时间
	Severity    AlertSeverity `json:"severity"`  // 严重级别
	Enabled     bool          `json:"enabled"`   // 是否启用
}

// AlertSeverity 告警严重级别
type AlertSeverity string

const (
	SeverityLow      AlertSeverity = "low"
	SeverityMedium   AlertSeverity = "medium"
	SeverityHigh     AlertSeverity = "high"
	SeverityCritical AlertSeverity = "critical"
)

// AlertNotifier 告警通知器接口
type AlertNotifier interface {
	SendAlert(alert *Alert) error
}

// Alert 告警信息
type Alert struct {
	RuleName    string            `json:"ruleName"`
	Severity    AlertSeverity     `json:"severity"`
	Message     string            `json:"message"`
	Timestamp   int64             `json:"timestamp"`
	NodeID      string            `json:"nodeId"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]any    `json:"annotations"`
}

// NewAlertManager 创建告警管理器
func NewAlertManager(notifier AlertNotifier) *AlertManager {
	return &AlertManager{
		rules:    make([]AlertRule, 0),
		notifier: notifier,
	}
}

// AddRule 添加告警规则
func (am *AlertManager) AddRule(rule AlertRule) {
	am.rules = append(am.rules, rule)
}

// EvaluateRules 评估告警规则
func (am *AlertManager) EvaluateRules(metrics *SystemMetrics) []*Alert {
	var alerts []*Alert

	for _, rule := range am.rules {
		if !rule.Enabled {
			continue
		}

		if am.evaluateCondition(rule, metrics) {
			alert := &Alert{
				RuleName:  rule.Name,
				Severity:  rule.Severity,
				Message:   rule.Description,
				Timestamp: time.Now().Unix(),
				NodeID:    metrics.NodeID,
				Labels: map[string]string{
					"rule":     rule.Name,
					"nodeId":   metrics.NodeID,
					"severity": string(rule.Severity),
				},
				Annotations: map[string]any{
					"threshold":    rule.Threshold,
					"currentValue": am.getCurrentValue(rule, metrics),
					"description":  rule.Description,
				},
			}

			alerts = append(alerts, alert)

			if am.notifier != nil {
				if err := am.notifier.SendAlert(alert); err != nil {
					log.Error().Err(err).Str("rule", rule.Name).Msg("failed to send alert")
				}
			}
		}
	}

	return alerts
}

// evaluateCondition 评估条件
func (am *AlertManager) evaluateCondition(rule AlertRule, metrics *SystemMetrics) bool {
	currentValue := am.getCurrentValue(rule, metrics)

	switch rule.Condition {
	case "greater_than":
		return currentValue > rule.Threshold
	case "less_than":
		return currentValue < rule.Threshold
	case "equals":
		return currentValue == rule.Threshold
	case "not_equals":
		return currentValue != rule.Threshold
	default:
		return false
	}
}

// getCurrentValue 获取当前值
func (am *AlertManager) getCurrentValue(rule AlertRule, metrics *SystemMetrics) float64 {
	// 简化实现，根据规则名称映射到指标值
	switch rule.Name {
	case "high_task_queue":
		return float64(metrics.PendingTasks)
	case "low_worker_availability":
		return float64(metrics.OnlineWorkers)
	case "high_error_rate":
		return metrics.ErrorRate
	case "high_worker_utilization":
		return metrics.WorkerUtilization
	default:
		return 0
	}
}

// DefaultAlertRules 默认告警规则
func DefaultAlertRules() []AlertRule {
	return []AlertRule{
		{
			Name:        "high_task_queue",
			Description: "Task queue length is too high",
			Condition:   "greater_than",
			Threshold:   1000,
			Duration:    5 * time.Minute,
			Severity:    SeverityHigh,
			Enabled:     true,
		},
		{
			Name:        "low_worker_availability",
			Description: "Available worker count is too low",
			Condition:   "less_than",
			Threshold:   1,
			Duration:    2 * time.Minute,
			Severity:    SeverityCritical,
			Enabled:     true,
		},
		{
			Name:        "high_error_rate",
			Description: "Task error rate is too high",
			Condition:   "greater_than",
			Threshold:   10.0, // 10%
			Duration:    3 * time.Minute,
			Severity:    SeverityMedium,
			Enabled:     true,
		},
		{
			Name:        "high_worker_utilization",
			Description: "Worker utilization is too high",
			Condition:   "greater_than",
			Threshold:   90.0, // 90%
			Duration:    10 * time.Minute,
			Severity:    SeverityMedium,
			Enabled:     true,
		},
	}
}

// LogAlertNotifier 日志告警通知器
type LogAlertNotifier struct{}

// SendAlert 发送告警到日志
func (n *LogAlertNotifier) SendAlert(alert *Alert) error {
	log.Warn().
		Str("rule", alert.RuleName).
		Str("severity", string(alert.Severity)).
		Str("nodeId", alert.NodeID).
		Str("message", alert.Message).
		Interface("annotations", alert.Annotations).
		Msg("alert triggered")

	return nil
}

// 工具函数：创建完整的监控系统
func SetupMonitoring(scheduler Scheduler, client redis.Cmdable, keys *RedisKeys, nodeID string) *MonitoringSystem {
	return &MonitoringSystem{
		MetricsCollector:   NewMetricsCollector(client, keys, nodeID, 30*time.Second),
		HealthChecker:      NewHealthChecker(scheduler, 30*time.Second),
		PerformanceMonitor: NewPerformanceMonitor(client, keys),
		AlertManager:       NewAlertManager(&LogAlertNotifier{}),
	}
}

// MonitoringSystem 监控系统
type MonitoringSystem struct {
	MetricsCollector   *MetricsCollector
	HealthChecker      *HealthChecker
	PerformanceMonitor *PerformanceMonitor
	AlertManager       *AlertManager
}

// Start 启动监控系统
func (ms *MonitoringSystem) Start(ctx context.Context) {
	// 添加默认告警规则
	for _, rule := range DefaultAlertRules() {
		ms.AlertManager.AddRule(rule)
	}

	// 启动各个组件
	go ms.MetricsCollector.Start(ctx)
	go ms.HealthChecker.Start(ctx)

	log.Info().Msg("monitoring system started")
}

// Stop 停止监控系统
func (ms *MonitoringSystem) Stop() {
	ms.MetricsCollector.Stop()
	ms.HealthChecker.Stop()
	log.Info().Msg("monitoring system stopped")
}
