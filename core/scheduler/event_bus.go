package scheduler

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kochabonline/kit/log"
)

// EventPublisher 事件发布者接口
type EventPublisher interface {
	Publish(event *SchedulerEvent) error
	PublishWithTimeout(event *SchedulerEvent, timeout time.Duration) error
}

// EventSubscriber 事件订阅者接口
type EventSubscriber interface {
	Subscribe(eventType EventType, listener EventListener)
}

// EventBusManager 事件总线管理接口
type EventBusManager interface {
	EventPublisher
	EventSubscriber
	GetMetrics() EventBusMetrics
	GetBufferStatus() BufferStatus
	Stop() error
}

// MetricsProvider 指标提供者接口
type MetricsProvider interface {
	GetMetrics() EventBusMetrics
	GetBufferStatus() BufferStatus
}

// EventPriority 事件优先级
type EventPriority int

const (
	PriorityLow    EventPriority = iota + 1 // 低优先级
	PriorityNormal                          // 普通优先级
	PriorityHigh                            // 高优先级
)

// String 返回优先级的字符串表示
func (p EventPriority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityNormal:
		return "normal"
	case PriorityHigh:
		return "high"
	default:
		return "unknown"
	}
}

// BufferInfo 缓冲区信息
type BufferInfo struct {
	Length   int `json:"length"`
	Capacity int `json:"capacity"`
}

// BufferStatus 缓冲区状态
type BufferStatus struct {
	HighPriority   BufferInfo     `json:"high_priority"`
	NormalPriority BufferInfo     `json:"normal_priority"`
	LowPriority    BufferInfo     `json:"low_priority"`
	Overflow       BufferInfo     `json:"overflow"`
	Expanding      bool           `json:"expanding"`
	Config         *DynamicConfig `json:"config"`
}

// EventBusConfig 事件总线配置
type EventBusConfig struct {
	BufferSize      int
	DynamicConfig   *DynamicConfig
	EnableMetrics   bool
	EnableDedup     bool
	MetricsInterval time.Duration
	PublishTimeout  time.Duration // Publish方法的默认超时时间
}

// EventBus 事件总线
type EventBus struct {
	// 读写锁保护listeners
	mu        sync.RWMutex
	listeners map[EventType][]EventListener

	// 缓冲区组，按优先级分组
	buffers struct {
		high     chan *SchedulerEvent // 高优先级缓冲区
		normal   chan *SchedulerEvent // 普通优先级缓冲区
		low      chan *SchedulerEvent // 低优先级缓冲区
		overflow chan *SchedulerEvent // 溢出缓冲区
	}

	// 动态扩容相关，使用原子操作优化
	config *DynamicConfig
	state  struct {
		expanding int32      // 原子操作：是否正在扩容
		expandMu  sync.Mutex // 扩容操作锁
	}

	// 去重和生命周期管理
	dedupMap sync.Map // 去重映射
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup

	// 指标，使用原子操作优化性能
	metrics *EventBusMetrics

	// 发布超时时间
	publishTimeout time.Duration
}

// EventBusMetrics 事件总线指标
type EventBusMetrics struct {
	// 使用原子操作的计数器
	publishedEvents int64 // 发布事件数
	droppedEvents   int64 // 丢弃事件数
	processedEvents int64 // 处理事件数
	duplicateEvents int64 // 重复事件数
	expansionCount  int64 // 扩容次数
	lastEventTime   int64 // 最后事件时间

	// 需要锁保护的复杂数据
	mu                sync.RWMutex
	bufferUtilization float64 // 缓冲区利用率
	currentCapacity   int64   // 当前总容量
	maxCapacity       int64   // 最大容量限制
}

// GetPublishedEvents 获取发布事件数
func (m *EventBusMetrics) GetPublishedEvents() int64 {
	return atomic.LoadInt64(&m.publishedEvents)
}

// IncrementPublishedEvents 增加发布事件数
func (m *EventBusMetrics) IncrementPublishedEvents() {
	atomic.AddInt64(&m.publishedEvents, 1)
	atomic.StoreInt64(&m.lastEventTime, time.Now().Unix())
}

// GetDroppedEvents 获取丢弃事件数
func (m *EventBusMetrics) GetDroppedEvents() int64 {
	return atomic.LoadInt64(&m.droppedEvents)
}

// IncrementDroppedEvents 增加丢弃事件数
func (m *EventBusMetrics) IncrementDroppedEvents() {
	atomic.AddInt64(&m.droppedEvents, 1)
}

// GetProcessedEvents 获取处理事件数
func (m *EventBusMetrics) GetProcessedEvents() int64 {
	return atomic.LoadInt64(&m.processedEvents)
}

// IncrementProcessedEvents 增加处理事件数
func (m *EventBusMetrics) IncrementProcessedEvents() {
	atomic.AddInt64(&m.processedEvents, 1)
}

// GetDuplicateEvents 获取重复事件数
func (m *EventBusMetrics) GetDuplicateEvents() int64 {
	return atomic.LoadInt64(&m.duplicateEvents)
}

// IncrementDuplicateEvents 增加重复事件数
func (m *EventBusMetrics) IncrementDuplicateEvents() {
	atomic.AddInt64(&m.duplicateEvents, 1)
}

// GetExpansionCount 获取扩容次数
func (m *EventBusMetrics) GetExpansionCount() int64 {
	return atomic.LoadInt64(&m.expansionCount)
}

// IncrementExpansionCount 增加扩容次数
func (m *EventBusMetrics) IncrementExpansionCount() {
	atomic.AddInt64(&m.expansionCount, 1)
}

// SetBufferUtilization 设置缓冲区利用率
func (m *EventBusMetrics) SetBufferUtilization(utilization float64) {
	m.mu.Lock()
	m.bufferUtilization = utilization
	m.mu.Unlock()
}

// GetBufferUtilization 获取缓冲区利用率
func (m *EventBusMetrics) GetBufferUtilization() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.bufferUtilization
}

// SetCurrentCapacity 设置当前容量
func (m *EventBusMetrics) SetCurrentCapacity(capacity int64) {
	m.mu.Lock()
	m.currentCapacity = capacity
	m.mu.Unlock()
}

// GetCurrentCapacity 获取当前容量
func (m *EventBusMetrics) GetCurrentCapacity() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentCapacity
}

// ToSnapshot 转换为快照
func (m *EventBusMetrics) ToSnapshot() EventBusMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return EventBusMetrics{
		publishedEvents:   atomic.LoadInt64(&m.publishedEvents),
		droppedEvents:     atomic.LoadInt64(&m.droppedEvents),
		processedEvents:   atomic.LoadInt64(&m.processedEvents),
		duplicateEvents:   atomic.LoadInt64(&m.duplicateEvents),
		expansionCount:    atomic.LoadInt64(&m.expansionCount),
		lastEventTime:     atomic.LoadInt64(&m.lastEventTime),
		bufferUtilization: m.bufferUtilization,
		currentCapacity:   m.currentCapacity,
		maxCapacity:       m.maxCapacity,
	}
}

// DynamicConfig 动态扩容配置
type DynamicConfig struct {
	InitialSize     int           // 初始缓冲区大小
	MaxSize         int           // 最大缓冲区大小
	ExpandThreshold float64       // 扩容阈值（利用率）
	ExpandFactor    float64       // 扩容因子
	ShrinkThreshold float64       // 收缩阈值
	ShrinkFactor    float64       // 收缩因子
	CheckInterval   time.Duration // 检查间隔
	OverflowBuffer  int           // 溢出缓冲区大小
}

// EventDeduplication 事件去重信息
type EventDeduplication struct {
	timestamp time.Time
	count     int
}

// EventListener 事件监听器
type EventListener func(ctx context.Context, event *SchedulerEvent) error

// NewEventBus 创建事件总线
func NewEventBus(bufferSize int) *EventBus {
	return NewEventBusWithConfig(&EventBusConfig{
		BufferSize:     bufferSize,
		PublishTimeout: 100 * time.Millisecond,
		DynamicConfig: &DynamicConfig{
			InitialSize:     bufferSize,
			MaxSize:         bufferSize * 10, // 最大可扩容到10倍
			ExpandThreshold: 0.8,             // 80%时开始扩容
			ExpandFactor:    1.5,             // 扩容1.5倍
			ShrinkThreshold: 0.3,             // 30%以下时收缩
			ShrinkFactor:    0.7,             // 收缩到70%
			CheckInterval:   2 * time.Second, // 每2秒检查一次
			OverflowBuffer:  bufferSize / 2,  // 溢出缓冲区为初始大小的一半
		},
	})
}

// NewEventBusWithConfig 使用配置创建事件总线
// 只支持 *EventBusConfig 配置类型
func NewEventBusWithConfig(config *EventBusConfig) *EventBus {
	ctx, cancel := context.WithCancel(context.Background())

	dynamicConfig := config.DynamicConfig
	bufferSize := config.BufferSize
	defaultTimeout := config.PublishTimeout
	if defaultTimeout <= 0 {
		defaultTimeout = 100 * time.Millisecond // 默认100ms
	}

	// 分配不同优先级的缓冲区大小
	highPrioritySize := bufferSize / 4   // 25% 高优先级
	normalPrioritySize := bufferSize / 2 // 50% 普通优先级
	lowPrioritySize := bufferSize / 4    // 25% 低优先级

	bus := &EventBus{
		listeners:      make(map[EventType][]EventListener),
		config:         dynamicConfig,
		ctx:            ctx,
		cancel:         cancel,
		publishTimeout: defaultTimeout,
		metrics: &EventBusMetrics{
			maxCapacity: int64(dynamicConfig.MaxSize),
		},
	}

	// 初始化缓冲区
	bus.buffers.high = make(chan *SchedulerEvent, highPrioritySize)
	bus.buffers.normal = make(chan *SchedulerEvent, normalPrioritySize)
	bus.buffers.low = make(chan *SchedulerEvent, lowPrioritySize)
	bus.buffers.overflow = make(chan *SchedulerEvent, dynamicConfig.OverflowBuffer)

	// 设置初始容量
	bus.metrics.SetCurrentCapacity(int64(bufferSize))

	// 启动事件处理循环
	bus.wg.Add(1)
	go bus.processEvents()

	// 启动指标更新和动态扩容检查
	bus.wg.Add(1)
	go bus.updateMetricsAndResize()

	return bus
}

// Subscribe 订阅事件
func (eb *EventBus) Subscribe(eventType EventType, listener EventListener) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.listeners[eventType] = append(eb.listeners[eventType], listener)

	log.Debug().
		Str("eventType", eventType.String()).
		Msg("event listener subscribed")
}

// getEventPriority 获取事件优先级
func (eb *EventBus) getEventPriority(eventType EventType) EventPriority {
	switch eventType {
	case EventSchedulerStarted, EventSchedulerStopped, EventLeaderElected, EventLeaderLost:
		return PriorityHigh
	case EventWorkerLeft, EventWorkerOffline, EventTaskFailed:
		return PriorityHigh
	case EventTaskSubmitted, EventTaskScheduled, EventWorkerJoined:
		return PriorityNormal
	case EventWorkerBusy, EventWorkerIdle, EventTaskStarted, EventTaskCompleted:
		return PriorityLow
	default:
		return PriorityNormal
	}
}

// generateDedupKey 生成去重键
func (eb *EventBus) generateDedupKey(event *SchedulerEvent) string {
	// 对于工作节点状态变更事件，使用工作节点ID+事件类型作为去重键
	if event.WorkerID != "" && (event.Type == EventWorkerBusy || event.Type == EventWorkerIdle) {
		return fmt.Sprintf("%s:%s", event.Type.String(), event.WorkerID)
	}
	// 其他事件不去重
	return ""
}

// Publish 发布事件（使用默认超时）
func (eb *EventBus) Publish(event *SchedulerEvent) error {
	return eb.PublishWithTimeout(event, eb.publishTimeout)
}

// PublishWithTimeout 带自定义超时的发布事件
func (eb *EventBus) PublishWithTimeout(event *SchedulerEvent, timeout time.Duration) error {
	// 更新发布事件计数
	eb.metrics.IncrementPublishedEvents()

	// 检查去重
	dedupKey := eb.generateDedupKey(event)
	if dedupKey != "" {
		if dedupInfo, exists := eb.dedupMap.Load(dedupKey); exists {
			dedup := dedupInfo.(*EventDeduplication)
			// 如果在1秒内有相同事件，则去重
			if time.Since(dedup.timestamp) < time.Second {
				dedup.count++
				eb.metrics.IncrementDuplicateEvents()
				return nil // 静默去重，不返回错误
			}
		}
		// 更新或创建去重信息
		eb.dedupMap.Store(dedupKey, &EventDeduplication{
			timestamp: time.Now(),
			count:     1,
		})
	}

	// 根据优先级选择缓冲区
	priority := eb.getEventPriority(event.Type)

	// 尝试发布到主缓冲区
	if err := eb.tryPublishToMainBuffer(event, priority, timeout); err == nil {
		return nil
	}

	// 主缓冲区满了，尝试使用溢出缓冲区
	log.Debug().
		Str("eventType", event.Type.String()).
		Msg("main buffer full, trying overflow buffer")

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case eb.buffers.overflow <- event:
		// 触发动态扩容检查
		go eb.triggerExpansionCheck()
		return nil
	case <-eb.ctx.Done():
		return fmt.Errorf("event bus is closed")
	case <-timer.C:
		// 更新丢弃事件计数
		eb.metrics.IncrementDroppedEvents()

		log.Error().
			Str("eventType", event.Type.String()).
			Msg("both main and overflow buffers are full, dropping event")
		return fmt.Errorf("event buffer is full")
	}
}

// tryPublishToMainBuffer 尝试发布到主缓冲区
func (eb *EventBus) tryPublishToMainBuffer(event *SchedulerEvent, priority EventPriority, timeout time.Duration) error {
	var targetBuffer chan *SchedulerEvent
	var bufferName string

	switch priority {
	case PriorityHigh:
		targetBuffer = eb.buffers.high
		bufferName = "high"
	case PriorityNormal:
		targetBuffer = eb.buffers.normal
		bufferName = "normal"
	case PriorityLow:
		targetBuffer = eb.buffers.low
		bufferName = "low"
	}

	timer := time.NewTimer(timeout / 2) // 给主缓冲区一半的超时时间
	defer timer.Stop()

	select {
	case targetBuffer <- event:
		return nil
	case <-eb.ctx.Done():
		return fmt.Errorf("event bus is closed")
	case <-timer.C:
		log.Debug().
			Int("bufferLength", len(targetBuffer)).
			Int("bufferCapacity", cap(targetBuffer)).
			Str("priority", bufferName).
			Str("eventType", event.Type.String()).
			Msg("main buffer timeout, will try overflow buffer")
		return fmt.Errorf("main buffer timeout")
	}
}

// processEvents 处理事件
func (eb *EventBus) processEvents() {
	defer eb.wg.Done()

	for {
		select {
		case <-eb.ctx.Done():
			return
		// 最优先处理溢出缓冲区的事件
		case event := <-eb.buffers.overflow:
			eb.handleEvent(event)
		// 然后处理高优先级事件
		case event := <-eb.buffers.high:
			eb.handleEvent(event)
		default:
			// 如果没有高优先级和溢出事件，处理普通和低优先级事件
			select {
			case <-eb.ctx.Done():
				return
			case event := <-eb.buffers.overflow:
				eb.handleEvent(event)
			case event := <-eb.buffers.high:
				eb.handleEvent(event)
			case event := <-eb.buffers.normal:
				eb.handleEvent(event)
			case event := <-eb.buffers.low:
				eb.handleEvent(event)
			}
		}
	}
}

// handleEvent 处理单个事件
func (eb *EventBus) handleEvent(event *SchedulerEvent) {
	eb.mu.RLock()
	listeners := eb.listeners[event.Type]
	eb.mu.RUnlock()

	if len(listeners) == 0 {
		log.Debug().
			Str("eventType", event.Type.String()).
			Msg("no listeners for event type")
		return
	}

	// 并发处理所有监听器
	var wg sync.WaitGroup
	for _, listener := range listeners {
		wg.Add(1)
		go func(l EventListener) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Error().
						Any("panic", r).
						Str("eventType", event.Type.String()).
						Msg("event listener panic")
				}
			}()

			if err := l(eb.ctx, event); err != nil {
				log.Error().
					Err(err).
					Str("eventType", event.Type.String()).
					Msg("event listener error")
			}
		}(listener)
	}
	wg.Wait()

	// 更新处理事件计数
	eb.metrics.IncrementProcessedEvents()
}

// updateMetricsAndResize 定期更新指标并检查是否需要动态扩容
func (eb *EventBus) updateMetricsAndResize() {
	defer eb.wg.Done()

	ticker := time.NewTicker(eb.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-eb.ctx.Done():
			return
		case <-ticker.C:
			// 更新缓冲区利用率
			eb.updateBufferUtilization()

			// 检查是否需要扩容或收缩
			eb.checkAndResize()

			// 清理过期的去重记录
			eb.cleanupDedupMap()
		}
	}
}

// updateBufferUtilization 更新缓冲区利用率
func (eb *EventBus) updateBufferUtilization() {
	highLen := len(eb.buffers.high)
	normalLen := len(eb.buffers.normal)
	lowLen := len(eb.buffers.low)
	overflowLen := len(eb.buffers.overflow)

	highCap := cap(eb.buffers.high)
	normalCap := cap(eb.buffers.normal)
	lowCap := cap(eb.buffers.low)
	overflowCap := cap(eb.buffers.overflow)

	totalUsed := highLen + normalLen + lowLen + overflowLen
	totalCap := highCap + normalCap + lowCap + overflowCap

	if totalCap > 0 {
		eb.metrics.SetBufferUtilization(float64(totalUsed) / float64(totalCap))
	}
	eb.metrics.SetCurrentCapacity(int64(totalCap))
}

// checkAndResize 检查并执行动态扩容或收缩
func (eb *EventBus) checkAndResize() {
	utilization := eb.metrics.GetBufferUtilization()
	currentCap := eb.metrics.GetCurrentCapacity()

	// 检查是否需要扩容
	if utilization >= eb.config.ExpandThreshold && currentCap < int64(eb.config.MaxSize) {
		eb.expandBuffers()
	} else if utilization <= eb.config.ShrinkThreshold && currentCap > int64(eb.config.InitialSize) {
		// 检查是否需要收缩
		eb.shrinkBuffers()
	}
}

// expandBuffers 扩容缓冲区
func (eb *EventBus) expandBuffers() {
	eb.state.expandMu.Lock()
	defer eb.state.expandMu.Unlock()

	if atomic.LoadInt32(&eb.state.expanding) == 1 {
		return // 已在扩容中
	}
	atomic.StoreInt32(&eb.state.expanding, 1)
	defer atomic.StoreInt32(&eb.state.expanding, 0)

	log.Info().
		Float64("utilization", eb.metrics.GetBufferUtilization()).
		Int64("currentCapacity", eb.metrics.GetCurrentCapacity()).
		Msg("expanding event bus buffers")

	// 计算新的容量
	newHighCap := int(float64(cap(eb.buffers.high)) * eb.config.ExpandFactor)
	newNormalCap := int(float64(cap(eb.buffers.normal)) * eb.config.ExpandFactor)
	newLowCap := int(float64(cap(eb.buffers.low)) * eb.config.ExpandFactor)

	// 限制最大容量
	totalNewCap := newHighCap + newNormalCap + newLowCap
	if int64(totalNewCap) > int64(eb.config.MaxSize) {
		ratio := float64(eb.config.MaxSize) / float64(totalNewCap)
		newHighCap = int(float64(newHighCap) * ratio)
		newNormalCap = int(float64(newNormalCap) * ratio)
		newLowCap = int(float64(newLowCap) * ratio)
	}

	// 执行无损扩容
	eb.replaceBuffers(newHighCap, newNormalCap, newLowCap)

	// 更新指标
	eb.metrics.IncrementExpansionCount()

	log.Info().
		Int("newHighCap", newHighCap).
		Int("newNormalCap", newNormalCap).
		Int("newLowCap", newLowCap).
		Msg("event bus buffers expanded successfully")
}

// shrinkBuffers 收缩缓冲区
func (eb *EventBus) shrinkBuffers() {
	eb.state.expandMu.Lock()
	defer eb.state.expandMu.Unlock()

	if atomic.LoadInt32(&eb.state.expanding) == 1 {
		return // 扩容中不收缩
	}

	log.Debug().
		Float64("utilization", eb.metrics.GetBufferUtilization()).
		Int64("currentCapacity", eb.metrics.GetCurrentCapacity()).
		Msg("considering shrinking event bus buffers")

	// 计算新的容量
	newHighCap := int(float64(cap(eb.buffers.high)) * eb.config.ShrinkFactor)
	newNormalCap := int(float64(cap(eb.buffers.normal)) * eb.config.ShrinkFactor)
	newLowCap := int(float64(cap(eb.buffers.low)) * eb.config.ShrinkFactor)

	// 确保不小于初始大小
	minHighCap := eb.config.InitialSize / 4
	minNormalCap := eb.config.InitialSize / 2
	minLowCap := eb.config.InitialSize / 4

	if newHighCap < minHighCap {
		newHighCap = minHighCap
	}
	if newNormalCap < minNormalCap {
		newNormalCap = minNormalCap
	}
	if newLowCap < minLowCap {
		newLowCap = minLowCap
	}

	// 只有在实际会收缩时才执行
	if newHighCap < cap(eb.buffers.high) || newNormalCap < cap(eb.buffers.normal) || newLowCap < cap(eb.buffers.low) {
		eb.replaceBuffers(newHighCap, newNormalCap, newLowCap)

		log.Debug().
			Int("newHighCap", newHighCap).
			Int("newNormalCap", newNormalCap).
			Int("newLowCap", newLowCap).
			Msg("event bus buffers shrunk successfully")
	}
}

// replaceBuffers 无损替换缓冲区
func (eb *EventBus) replaceBuffers(newHighCap, newNormalCap, newLowCap int) {
	// 创建新的缓冲区
	newHighBuf := make(chan *SchedulerEvent, newHighCap)
	newNormalBuf := make(chan *SchedulerEvent, newNormalCap)
	newLowBuf := make(chan *SchedulerEvent, newLowCap)

	// 暂停新事件的处理，将现有事件迁移到新缓冲区
	eb.mu.Lock()
	defer eb.mu.Unlock()

	// 迁移高优先级缓冲区
	eb.migrateBuffer(eb.buffers.high, newHighBuf, "high")
	// 迁移普通优先级缓冲区
	eb.migrateBuffer(eb.buffers.normal, newNormalBuf, "normal")
	// 迁移低优先级缓冲区
	eb.migrateBuffer(eb.buffers.low, newLowBuf, "low")

	// 原子性地替换缓冲区
	eb.buffers.high = newHighBuf
	eb.buffers.normal = newNormalBuf
	eb.buffers.low = newLowBuf

	log.Debug().
		Int("highNewCap", newHighCap).
		Int("normalNewCap", newNormalCap).
		Int("lowNewCap", newLowCap).
		Msg("buffers replaced successfully")
}

// migrateBuffer 迁移缓冲区中的事件
func (eb *EventBus) migrateBuffer(oldBuf, newBuf chan *SchedulerEvent, bufferType string) {
	migratedCount := 0
	droppedCount := 0

	// 尽可能地迁移事件
	for {
		select {
		case event := <-oldBuf:
			select {
			case newBuf <- event:
				migratedCount++
			default:
				// 新缓冲区满了，只能丢弃事件
				droppedCount++
				atomic.AddInt64(&eb.metrics.droppedEvents, 1)

				log.Warn().
					Str("bufferType", bufferType).
					Str("eventType", event.Type.String()).
					Msg("event dropped during buffer migration")
			}
		default:
			// 旧缓冲区已空
			goto done
		}
	}

done:
	if migratedCount > 0 || droppedCount > 0 {
		log.Info().
			Str("bufferType", bufferType).
			Int("migrated", migratedCount).
			Int("dropped", droppedCount).
			Msg("buffer migration completed")
	}
}

// triggerExpansionCheck 触发扩容检查
func (eb *EventBus) triggerExpansionCheck() {
	// 异步触发，避免阻塞
	go func() {
		eb.updateBufferUtilization()
		eb.checkAndResize()
	}()
}

// cleanupDedupMap 清理过期的去重记录
func (eb *EventBus) cleanupDedupMap() {
	now := time.Now()
	eb.dedupMap.Range(func(key, value any) bool {
		dedup := value.(*EventDeduplication)
		// 清理5秒前的记录
		if now.Sub(dedup.timestamp) > 5*time.Second {
			eb.dedupMap.Delete(key)
		}
		return true
	})
}

// GetMetrics 获取事件总线指标
func (eb *EventBus) GetMetrics() EventBusMetrics {
	return EventBusMetrics{
		publishedEvents:   atomic.LoadInt64(&eb.metrics.publishedEvents),
		droppedEvents:     atomic.LoadInt64(&eb.metrics.droppedEvents),
		processedEvents:   atomic.LoadInt64(&eb.metrics.processedEvents),
		duplicateEvents:   atomic.LoadInt64(&eb.metrics.duplicateEvents),
		bufferUtilization: eb.metrics.GetBufferUtilization(),
		lastEventTime:     atomic.LoadInt64(&eb.metrics.lastEventTime),
		expansionCount:    atomic.LoadInt64(&eb.metrics.expansionCount),
		currentCapacity:   atomic.LoadInt64(&eb.metrics.currentCapacity),
		maxCapacity:       atomic.LoadInt64(&eb.metrics.maxCapacity),
	}
}

// GetBufferStatus 获取当前缓冲区状态
func (eb *EventBus) GetBufferStatus() map[string]any {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	return map[string]any{
		"high_priority": map[string]int{
			"length":   len(eb.buffers.high),
			"capacity": cap(eb.buffers.high),
		},
		"normal_priority": map[string]int{
			"length":   len(eb.buffers.normal),
			"capacity": cap(eb.buffers.normal),
		},
		"low_priority": map[string]int{
			"length":   len(eb.buffers.low),
			"capacity": cap(eb.buffers.low),
		},
		"overflow": map[string]int{
			"length":   len(eb.buffers.overflow),
			"capacity": cap(eb.buffers.overflow),
		},
		"expanding": eb.state.expanding,
		"config":    eb.config,
	}
}

// Stop 停止事件总线
func (eb *EventBus) Stop() error {
	eb.cancel()

	// 等待处理完成
	done := make(chan struct{})
	go func() {
		eb.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 正常停止不记录日志
	case <-time.After(10 * time.Second):
		log.Warn().Msg("timeout waiting for event bus to stop")
	}

	return nil
}

// EventProcessor 事件处理器
type EventProcessor struct {
	name      string
	scheduler *scheduler
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewEventProcessor 创建事件处理器
func NewEventProcessor(name string, scheduler *scheduler) *EventProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	return &EventProcessor{
		name:      name,
		scheduler: scheduler,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start 启动事件处理器
func (ep *EventProcessor) Start() error {
	switch ep.name {
	case "task":
		ep.wg.Add(1)
		go ep.processTaskEvents()
	case "worker":
		ep.wg.Add(1)
		go ep.processWorkerEvents()
	case "scheduler":
		ep.wg.Add(1)
		go ep.processSchedulerEvents()
	default:
		return fmt.Errorf("unknown processor type: %s", ep.name)
	}

	return nil
}

// Stop 停止事件处理器
func (ep *EventProcessor) Stop() error {
	ep.cancel()

	done := make(chan struct{})
	go func() {
		ep.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 只在超时时记录日志，正常停止不记录
	case <-time.After(10 * time.Second):
		log.Warn().Str("processor", ep.name).Msg("timeout stopping event processor")
	}

	return nil
}

// processTaskEvents 处理任务事件
func (ep *EventProcessor) processTaskEvents() {
	defer ep.wg.Done()

	// 注册任务事件监听器
	ep.scheduler.eventBus.Subscribe(EventTaskSubmitted, ep.handleTaskSubmitted)
	ep.scheduler.eventBus.Subscribe(EventTaskScheduled, ep.handleTaskScheduled)
	ep.scheduler.eventBus.Subscribe(EventTaskStarted, ep.handleTaskStarted)
	ep.scheduler.eventBus.Subscribe(EventTaskCompleted, ep.handleTaskCompleted)
	ep.scheduler.eventBus.Subscribe(EventTaskFailed, ep.handleTaskFailed)
	ep.scheduler.eventBus.Subscribe(EventTaskCanceled, ep.handleTaskCanceled)

	<-ep.ctx.Done()
}

// processWorkerEvents 处理工作节点事件
func (ep *EventProcessor) processWorkerEvents() {
	defer ep.wg.Done()

	// 注册工作节点事件监听器
	ep.scheduler.eventBus.Subscribe(EventWorkerJoined, ep.handleWorkerJoined)
	ep.scheduler.eventBus.Subscribe(EventWorkerLeft, ep.handleWorkerLeft)
	ep.scheduler.eventBus.Subscribe(EventWorkerOnline, ep.handleWorkerOnline)
	ep.scheduler.eventBus.Subscribe(EventWorkerOffline, ep.handleWorkerOffline)
	ep.scheduler.eventBus.Subscribe(EventWorkerBusy, ep.handleWorkerBusy)
	ep.scheduler.eventBus.Subscribe(EventWorkerIdle, ep.handleWorkerIdle)

	<-ep.ctx.Done()
}

// processSchedulerEvents 处理调度器事件
func (ep *EventProcessor) processSchedulerEvents() {
	defer ep.wg.Done()

	// 注册调度器事件监听器
	ep.scheduler.eventBus.Subscribe(EventSchedulerStarted, ep.handleSchedulerStarted)
	ep.scheduler.eventBus.Subscribe(EventSchedulerStopped, ep.handleSchedulerStopped)
	ep.scheduler.eventBus.Subscribe(EventLeaderElected, ep.handleLeaderElected)
	ep.scheduler.eventBus.Subscribe(EventLeaderLost, ep.handleLeaderLost)

	<-ep.ctx.Done()
}

// 任务事件处理方法
func (ep *EventProcessor) handleTaskSubmitted(ctx context.Context, event *SchedulerEvent) error {
	log.Info().
		Str("taskId", event.TaskID).
		Msg("task submitted")

	// 如果是Leader，异步触发任务调度
	if ep.scheduler.IsLeader() {
		go func() {
			scheduleCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := ep.scheduler.scheduleTask(scheduleCtx, event.TaskID); err != nil {
				log.Error().
					Err(err).
					Str("taskId", event.TaskID).
					Msg("failed to schedule task")
			}
		}()
	}

	return nil
}

func (ep *EventProcessor) handleTaskScheduled(ctx context.Context, event *SchedulerEvent) error {
	log.Info().
		Str("taskId", event.TaskID).
		Str("workerId", event.WorkerID).
		Msg("task scheduled")

	// 更新指标
	ep.scheduler.updateMetrics(func(m *SchedulerMetrics) {
		m.TasksPending--
		m.TasksRunning++
		m.TotalScheduleCount++
		m.LastScheduleTime = currentTimestamp()
	})

	return nil
}

func (ep *EventProcessor) handleTaskStarted(ctx context.Context, event *SchedulerEvent) error {
	log.Debug().
		Str("taskId", event.TaskID).
		Str("workerId", event.WorkerID).
		Msg("task started")

	return nil
}

func (ep *EventProcessor) handleTaskCompleted(ctx context.Context, event *SchedulerEvent) error {
	log.Debug().
		Str("taskId", event.TaskID).
		Str("workerId", event.WorkerID).
		Msg("task completed")

	// 更新指标
	ep.scheduler.updateMetrics(func(m *SchedulerMetrics) {
		m.TasksRunning--
		m.TasksCompleted++
	})

	return nil
}

func (ep *EventProcessor) handleTaskFailed(ctx context.Context, event *SchedulerEvent) error {
	log.Error().
		Str("taskId", event.TaskID).
		Str("workerId", event.WorkerID).
		Msg("task failed")

	// 更新指标
	ep.scheduler.updateMetrics(func(m *SchedulerMetrics) {
		m.TasksRunning--
		m.TasksFailed++
	})

	return nil
}

func (ep *EventProcessor) handleTaskCanceled(ctx context.Context, event *SchedulerEvent) error {
	log.Info().
		Str("taskId", event.TaskID).
		Msg("task canceled")

	// 更新指标
	ep.scheduler.updateMetrics(func(m *SchedulerMetrics) {
		m.TasksCanceled++
	})

	return nil
}

// 工作节点事件处理方法
func (ep *EventProcessor) handleWorkerJoined(ctx context.Context, event *SchedulerEvent) error {
	log.Info().
		Str("workerId", event.WorkerID).
		Msg("worker joined")

	// 更新指标
	ep.scheduler.updateMetrics(func(m *SchedulerMetrics) {
		m.WorkersTotal++
		m.WorkersOnline++
	})

	return nil
}

func (ep *EventProcessor) handleWorkerLeft(ctx context.Context, event *SchedulerEvent) error {
	log.Info().
		Str("workerId", event.WorkerID).
		Msg("worker left")

	// 更新指标
	ep.scheduler.updateMetrics(func(m *SchedulerMetrics) {
		m.WorkersTotal--
		m.WorkersOnline--
	})

	// 如果是Leader，异步重新调度该工作节点的任务
	if ep.scheduler.IsLeader() {
		go func() {
			rescheduleCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := ep.scheduler.rescheduleWorkerTasks(rescheduleCtx, event.WorkerID); err != nil {
				log.Error().
					Err(err).
					Str("workerId", event.WorkerID).
					Msg("failed to reschedule worker tasks")
			}
		}()
	}

	return nil
}

func (ep *EventProcessor) handleWorkerOnline(ctx context.Context, event *SchedulerEvent) error {
	log.Debug().
		Str("workerId", event.WorkerID).
		Msg("worker online")

	// 更新指标
	ep.scheduler.updateMetrics(func(m *SchedulerMetrics) {
		m.WorkersOffline--
		m.WorkersOnline++
	})

	return nil
}

func (ep *EventProcessor) handleWorkerOffline(ctx context.Context, event *SchedulerEvent) error {
	log.Warn().
		Str("workerId", event.WorkerID).
		Msg("worker offline")

	// 更新指标
	ep.scheduler.updateMetrics(func(m *SchedulerMetrics) {
		m.WorkersOnline--
		m.WorkersOffline++
	})

	// 如果是Leader，异步重新调度该工作节点的任务
	if ep.scheduler.IsLeader() {
		go func() {
			rescheduleCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := ep.scheduler.rescheduleWorkerTasks(rescheduleCtx, event.WorkerID); err != nil {
				log.Error().
					Err(err).
					Str("workerId", event.WorkerID).
					Msg("failed to reschedule worker tasks")
			}
		}()
	}

	return nil
}

func (ep *EventProcessor) handleWorkerBusy(ctx context.Context, event *SchedulerEvent) error {
	log.Debug().
		Str("workerId", event.WorkerID).
		Msg("worker busy")

	// 更新指标
	ep.scheduler.updateMetrics(func(m *SchedulerMetrics) {
		m.WorkersIdle--
		m.WorkersBusy++
	})

	return nil
}

func (ep *EventProcessor) handleWorkerIdle(ctx context.Context, event *SchedulerEvent) error {
	log.Debug().
		Str("workerId", event.WorkerID).
		Msg("worker idle")

	// 更新指标
	ep.scheduler.updateMetrics(func(m *SchedulerMetrics) {
		m.WorkersBusy--
		m.WorkersIdle++
	})

	return nil
}

// 调度器事件处理方法
func (ep *EventProcessor) handleSchedulerStarted(ctx context.Context, event *SchedulerEvent) error {
	return nil
}

func (ep *EventProcessor) handleSchedulerStopped(ctx context.Context, event *SchedulerEvent) error {
	// 不记录日志，避免重复信息
	return nil
}

func (ep *EventProcessor) handleLeaderElected(ctx context.Context, event *SchedulerEvent) error {
	log.Info().
		Str("nodeId", event.Data["nodeId"].(string)).
		Msg("leader elected")

	return nil
}

func (ep *EventProcessor) handleLeaderLost(ctx context.Context, event *SchedulerEvent) error {
	log.Warn().
		Str("nodeId", event.Data["nodeId"].(string)).
		Msg("leader lost")

	return nil
}
