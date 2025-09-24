package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kochabonline/kit/log"
	"github.com/redis/go-redis/v9"
)

// EventManager Redis事件管理器
type EventManager struct {
	client redis.UniversalClient // 使用UniversalClient接口支持Subscribe
	keys   *RedisKeys

	// Pub/Sub 管理
	pubsub *redis.PubSub

	// 事件回调
	mu        sync.RWMutex
	callbacks map[EventType][]EventCallback

	// Stream 消费者
	streamGroup    string
	streamConsumer string
	streamMaxLen   int64

	// 控制通道
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewEventManager 创建事件管理器
func NewEventManager(client redis.UniversalClient, keys *RedisKeys, streamGroup, streamConsumer string, streamMaxLen int64) *EventManager {
	return &EventManager{
		client:         client,
		keys:           keys,
		callbacks:      make(map[EventType][]EventCallback),
		streamGroup:    streamGroup,
		streamConsumer: streamConsumer,
		streamMaxLen:   streamMaxLen,
		stopChan:       make(chan struct{}),
	}
}

// Start 启动事件管理器
func (em *EventManager) Start(ctx context.Context) error {
	// 创建Stream消费者组
	if err := em.createConsumerGroup(ctx); err != nil {
		log.Error().Err(err).Msg("failed to create consumer group")
	}

	// 启动Pub/Sub监听器
	em.pubsub = em.client.Subscribe(ctx, em.keys.EventChannel)
	em.wg.Add(1)
	go em.runPubSubListener(ctx)

	// 启动Stream消费者
	em.wg.Add(1)
	go em.runStreamConsumer(ctx)

	log.Info().Str("channel", em.keys.EventChannel).Str("stream", em.keys.EventStream).Msg("event manager started")
	return nil
}

// Stop 停止事件管理器
func (em *EventManager) Stop() error {
	close(em.stopChan)

	if em.pubsub != nil {
		if err := em.pubsub.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close pubsub")
		}
	}

	em.wg.Wait()
	log.Info().Msg("event manager stopped")
	return nil
}

// RegisterCallback 注册事件回调
func (em *EventManager) RegisterCallback(eventType EventType, callback EventCallback) {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.callbacks[eventType] = append(em.callbacks[eventType], callback)
	log.Debug().Str("eventType", string(eventType)).Msg("event callback registered")
}

// PublishEvent 发布事件 (使用Pub/Sub + Stream双重机制)
func (em *EventManager) PublishEvent(ctx context.Context, event *SchedulerEvent) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	pipe := em.client.Pipeline()

	// 1. 发布到Pub/Sub频道 (实时通知)
	pipe.Publish(ctx, em.keys.EventChannel, eventJSON)

	// 2. 添加到Stream (持久化和可靠消费)
	streamValues := map[string]any{
		"type":      string(event.Type),
		"taskId":    event.TaskID,
		"workerId":  event.WorkerID,
		"nodeId":    event.NodeID,
		"timestamp": event.Timestamp,
		"data":      string(eventJSON),
	}

	// 使用MAXLEN限制Stream长度
	pipe.XAdd(ctx, &redis.XAddArgs{
		Stream: em.keys.EventStream,
		MaxLen: em.streamMaxLen,
		Approx: true, // 使用近似MAXLEN以提高性能
		Values: streamValues,
	})

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	log.Debug().Str("type", string(event.Type)).Str("taskId", event.TaskID).Msg("event published")
	return nil
}

// PublishTaskEvent 发布任务相关事件的便捷方法
func (em *EventManager) PublishTaskEvent(ctx context.Context, eventType EventType, task *Task, error string) error {
	event := &SchedulerEvent{
		Type:      eventType,
		Timestamp: time.Now().Unix(),
		Error:     error,
	}

	// 添加任务相关的数据
	if task != nil {
		event.TaskID = task.ID
		event.WorkerID = task.WorkerID
		event.Data = map[string]any{
			"priority": task.Priority.String(),
			"status":   task.Status.String(),
		}

		if eventType == EventTaskCompleted {
			event.Data["duration"] = task.CompletedAt - task.StartedAt
		}
	}

	return em.PublishEvent(ctx, event)
}

// PublishWorkerEvent 发布工作节点相关事件的便捷方法
func (em *EventManager) PublishWorkerEvent(ctx context.Context, eventType EventType, worker *Worker) error {
	event := &SchedulerEvent{
		Type:      eventType,
		Timestamp: time.Now().Unix(),
	}

	// 添加工作节点相关的数据
	if worker != nil {
		event.WorkerID = worker.ID
		event.Data = map[string]any{
			"address":      worker.Address,
			"status":       worker.Status.String(),
			"currentTasks": worker.CurrentTasks,
			"maxTasks":     worker.MaxTasks,
			"cpu":          worker.CPU,
			"memory":       worker.Memory,
			"load":         worker.Load,
		}
	}

	return em.PublishEvent(ctx, event)
}

// runPubSubListener 运行Pub/Sub监听器
func (em *EventManager) runPubSubListener(ctx context.Context) {
	defer em.wg.Done()

	ch := em.pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			return
		case <-em.stopChan:
			return
		case msg := <-ch:
			if msg == nil {
				continue
			}

			// 解析事件
			var event SchedulerEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				log.Error().Err(err).Str("payload", msg.Payload).Msg("failed to unmarshal event")
				continue
			}

			// 触发回调
			em.triggerCallbacks(ctx, &event)
		}
	}
}

// runStreamConsumer 运行Stream消费者
func (em *EventManager) runStreamConsumer(ctx context.Context) {
	defer em.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-em.stopChan:
			return
		default:
			// 从Stream读取消息
			messages, err := em.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    em.streamGroup,
				Consumer: em.streamConsumer,
				Streams:  []string{em.keys.EventStream, ">"},
				Count:    10,
				Block:    5 * time.Second,
			}).Result()

			if err != nil {
				if err != redis.Nil {
					log.Error().Err(err).Msg("failed to read from stream")
				}
				continue
			}

			// 处理消息
			for _, stream := range messages {
				for _, message := range stream.Messages {
					if err := em.processStreamMessage(ctx, message); err != nil {
						log.Error().Err(err).Str("messageId", message.ID).Msg("failed to process stream message")
						continue
					}

					// 确认消息
					if err := em.client.XAck(ctx, em.keys.EventStream, em.streamGroup, message.ID).Err(); err != nil {
						log.Error().Err(err).Str("messageId", message.ID).Msg("failed to ack stream message")
					}
				}
			}
		}
	}
}

// processStreamMessage 处理Stream消息
func (em *EventManager) processStreamMessage(ctx context.Context, message redis.XMessage) error {
	// 从消息中提取事件数据
	eventDataStr, ok := message.Values["data"].(string)
	if !ok {
		return fmt.Errorf("missing event data in message")
	}

	var event SchedulerEvent
	if err := json.Unmarshal([]byte(eventDataStr), &event); err != nil {
		return fmt.Errorf("failed to unmarshal event data: %w", err)
	}

	// 触发回调 (仅在Pub/Sub监听器未处理的情况下)
	// 这里可以添加去重逻辑，避免重复处理
	return nil
}

// triggerCallbacks 触发事件回调
func (em *EventManager) triggerCallbacks(ctx context.Context, event *SchedulerEvent) {
	em.mu.RLock()
	callbacks, exists := em.callbacks[event.Type]
	em.mu.RUnlock()

	if !exists || len(callbacks) == 0 {
		return
	}

	// 并发触发所有回调
	var wg sync.WaitGroup
	for _, callback := range callbacks {
		wg.Add(1)
		go func(cb EventCallback) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Error().Interface("panic", r).Str("eventType", string(event.Type)).Msg("event callback panicked")
				}
			}()

			// 设置超时上下文
			callbackCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			if err := cb(callbackCtx, event); err != nil {
				log.Error().Err(err).Str("eventType", string(event.Type)).Msg("event callback failed")
			}
		}(callback)
	}

	wg.Wait()
}

// createConsumerGroup 创建Stream消费者组
func (em *EventManager) createConsumerGroup(ctx context.Context) error {
	// 尝试创建消费者组
	err := em.client.XGroupCreateMkStream(ctx, em.keys.EventStream, em.streamGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	log.Info().Str("stream", em.keys.EventStream).Str("group", em.streamGroup).Msg("consumer group created/verified")
	return nil
}

// GetPendingMessages 获取待处理的Stream消息
func (em *EventManager) GetPendingMessages(ctx context.Context) ([]redis.XPendingExt, error) {
	pending, err := em.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: em.keys.EventStream,
		Group:  em.streamGroup,
		Start:  "-",
		End:    "+",
		Count:  100,
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("failed to get pending messages: %w", err)
	}

	return pending, nil
}

// ClaimStaleMessages 认领超时的Stream消息
func (em *EventManager) ClaimStaleMessages(ctx context.Context, minIdleTime time.Duration) error {
	// 获取待处理消息
	pending, err := em.GetPendingMessages(ctx)
	if err != nil {
		return err
	}

	var staleMessageIDs []string
	for _, msg := range pending {
		if msg.Idle >= minIdleTime {
			staleMessageIDs = append(staleMessageIDs, msg.ID)
		}
	}

	if len(staleMessageIDs) == 0 {
		return nil
	}

	// 认领超时消息
	claimed, err := em.client.XClaim(ctx, &redis.XClaimArgs{
		Stream:   em.keys.EventStream,
		Group:    em.streamGroup,
		Consumer: em.streamConsumer,
		Messages: staleMessageIDs,
		MinIdle:  minIdleTime,
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to claim stale messages: %w", err)
	}

	log.Info().Int("count", len(claimed)).Msg("claimed stale messages")

	// 处理认领的消息
	for _, message := range claimed {
		if err := em.processStreamMessage(ctx, message); err != nil {
			log.Error().Err(err).Str("messageId", message.ID).Msg("failed to process claimed message")
			continue
		}

		// 确认消息
		if err := em.client.XAck(ctx, em.keys.EventStream, em.streamGroup, message.ID).Err(); err != nil {
			log.Error().Err(err).Str("messageId", message.ID).Msg("failed to ack claimed message")
		}
	}

	return nil
}

// GetStreamInfo 获取Stream信息
func (em *EventManager) GetStreamInfo(ctx context.Context) (*redis.XInfoStream, error) {
	info, err := em.client.XInfoStream(ctx, em.keys.EventStream).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}

	return info, nil
}

// GetConsumerGroupInfo 获取消费者组信息
func (em *EventManager) GetConsumerGroupInfo(ctx context.Context) ([]redis.XInfoGroup, error) {
	groups, err := em.client.XInfoGroups(ctx, em.keys.EventStream).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer group info: %w", err)
	}

	return groups, nil
}

// redisPubSubManager Redis Pub/Sub管理器实现
type redisPubSubManager struct {
	client redis.UniversalClient
	pubsub *redis.PubSub
	mu     sync.RWMutex
}

// NewRedisPubSubManager 创建Redis Pub/Sub管理器
func NewRedisPubSubManager(client redis.UniversalClient) PubSubManager {
	return &redisPubSubManager{
		client: client,
	}
}

// Publish 发布消息
func (p *redisPubSubManager) Publish(ctx context.Context, channel string, message any) error {
	var payload string

	switch v := message.(type) {
	case string:
		payload = v
	case []byte:
		payload = string(v)
	default:
		data, err := json.Marshal(message)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}
		payload = string(data)
	}

	return p.client.Publish(ctx, channel, payload).Err()
}

// Subscribe 订阅消息
func (p *redisPubSubManager) Subscribe(ctx context.Context, channels ...string) <-chan *Message {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.pubsub = p.client.Subscribe(ctx, channels...)
	ch := p.pubsub.Channel()

	msgChan := make(chan *Message, 100)

	go func() {
		defer close(msgChan)
		for msg := range ch {
			if msg == nil {
				continue
			}

			message := &Message{
				Channel: msg.Channel,
				Pattern: msg.Pattern,
				Payload: msg.Payload,
			}

			select {
			case msgChan <- message:
			case <-ctx.Done():
				return
			}
		}
	}()

	return msgChan
}

// Unsubscribe 取消订阅
func (p *redisPubSubManager) Unsubscribe(ctx context.Context, channels ...string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.pubsub == nil {
		return nil
	}

	return p.pubsub.Unsubscribe(ctx, channels...)
}

// Close 关闭连接
func (p *redisPubSubManager) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.pubsub == nil {
		return nil
	}

	return p.pubsub.Close()
}

// redisStreamManager Redis Stream管理器实现
type redisStreamManager struct {
	client redis.Cmdable
}

// NewRedisStreamManager 创建Redis Stream管理器
func NewRedisStreamManager(client redis.Cmdable) StreamManager {
	return &redisStreamManager{
		client: client,
	}
}

// AddMessage 添加消息到Stream
func (s *redisStreamManager) AddMessage(ctx context.Context, stream string, values map[string]any) (string, error) {
	result, err := s.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Result()

	if err != nil {
		return "", fmt.Errorf("failed to add message to stream: %w", err)
	}

	return result, nil
}

// ReadGroup 消费者组读取消息
func (s *redisStreamManager) ReadGroup(ctx context.Context, group, consumer, stream, lastID string, count int64) ([]StreamMessage, error) {
	result, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, lastID},
		Count:    count,
		Block:    time.Second,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return []StreamMessage{}, nil
		}
		return nil, fmt.Errorf("failed to read from stream: %w", err)
	}

	var messages []StreamMessage
	for _, streamResult := range result {
		for _, message := range streamResult.Messages {
			values := make(map[string]string)
			for k, v := range message.Values {
				if str, ok := v.(string); ok {
					values[k] = str
				} else {
					values[k] = fmt.Sprintf("%v", v)
				}
			}

			messages = append(messages, StreamMessage{
				ID:     message.ID,
				Values: values,
			})
		}
	}

	return messages, nil
}

// CreateGroup 创建消费者组
func (s *redisStreamManager) CreateGroup(ctx context.Context, stream, group, startID string) error {
	err := s.client.XGroupCreateMkStream(ctx, stream, group, startID).Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	return nil
}

// AckMessage 确认消息
func (s *redisStreamManager) AckMessage(ctx context.Context, stream, group string, messageIDs ...string) error {
	return s.client.XAck(ctx, stream, group, messageIDs...).Err()
}
