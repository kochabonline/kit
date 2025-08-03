package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kochabonline/kit/log"
)

// EventBus 事件总线
type EventBus struct {
	mu        sync.RWMutex
	listeners map[EventType][]EventListener
	buffer    chan *SchedulerEvent
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// EventListener 事件监听器
type EventListener func(ctx context.Context, event *SchedulerEvent) error

// NewEventBus 创建事件总线
func NewEventBus(bufferSize int) *EventBus {
	ctx, cancel := context.WithCancel(context.Background())

	bus := &EventBus{
		listeners: make(map[EventType][]EventListener),
		buffer:    make(chan *SchedulerEvent, bufferSize),
		ctx:       ctx,
		cancel:    cancel,
	}

	// 启动事件处理循环
	bus.wg.Add(1)
	go bus.processEvents()

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

// Publish 发布事件
func (eb *EventBus) Publish(event *SchedulerEvent) error {
	select {
	case eb.buffer <- event:
		return nil
	case <-eb.ctx.Done():
		return fmt.Errorf("event bus is closed")
	default:
		return fmt.Errorf("event buffer is full")
	}
}

// processEvents 处理事件
func (eb *EventBus) processEvents() {
	defer eb.wg.Done()

	for {
		select {
		case <-eb.ctx.Done():
			return
		case event := <-eb.buffer:
			eb.handleEvent(event)
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

	// 如果是Leader，触发任务调度
	if ep.scheduler.IsLeader() {
		return ep.scheduler.scheduleTask(ctx, event.TaskID)
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

	// 如果是Leader，重新调度该工作节点的任务
	if ep.scheduler.IsLeader() {
		return ep.scheduler.rescheduleWorkerTasks(ctx, event.WorkerID)
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
