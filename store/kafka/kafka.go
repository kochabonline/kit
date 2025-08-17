package kafka

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
	"golang.org/x/sync/errgroup"
)

// Kafka 客户端封装，提供生产者和消费者的管理
type Kafka struct {
	config    *Config                  // 配置信息
	dialer    *kafka.Dialer            // 连接拨号器
	transport *kafka.Transport         // 传输配置
	producers map[string]*kafka.Writer // 生产者映射表，按主题存储
	consumers map[string]*kafka.Reader // 消费者映射表，按主题或主题-组合ID存储
	mu        sync.Mutex               // 互斥锁，保护并发访问
}

// Option 配置选项函数类型
type Option func(*Kafka)

// New 创建新的Kafka客户端实例
func New(config *Config, opts ...Option) (*Kafka, error) {
	k := &Kafka{
		config:    config,
		producers: make(map[string]*kafka.Writer),
		consumers: make(map[string]*kafka.Reader),
	}

	if err := k.config.init(); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(k)
	}

	k.dialer = k.createDialer()
	k.transport = k.createTransport()

	return k, nil
}

// mechanism 创建SASL认证机制
func (k *Kafka) mechanism() plain.Mechanism {
	return plain.Mechanism{
		Username: k.config.Username,
		Password: k.config.Password,
	}
}

// createDialer 创建Kafka连接拨号器
func (k *Kafka) createDialer() *kafka.Dialer {
	dialer := &kafka.Dialer{
		Timeout:   time.Duration(k.config.Timeout) * time.Second,
		DualStack: true,
	}

	if k.config.Username != "" && k.config.Password != "" {
		dialer.SASLMechanism = k.mechanism()
	}

	return dialer
}

// createTransport 创建Kafka传输配置
func (k *Kafka) createTransport() *kafka.Transport {
	transport := &kafka.Transport{}

	if k.config.Username != "" && k.config.Password != "" {
		transport.SASL = k.mechanism()
	}

	return transport
}

// Producer 获取指定主题的同步生产者，如果不存在则创建
func (k *Kafka) Producer(topic string) *kafka.Writer {
	k.mu.Lock()
	writer, exists := k.producers[topic]
	k.mu.Unlock()
	if exists {
		return writer
	}

	writer = &kafka.Writer{
		Addr:                   kafka.TCP(k.config.Brokers...),
		Topic:                  topic,
		Balancer:               k.config.balancer(),
		Transport:              k.transport,
		AllowAutoTopicCreation: k.config.AllowAutoTopicCreation,
	}

	k.mu.Lock()
	k.producers[topic] = writer
	k.mu.Unlock()

	return writer
}

// AsyncProducer 获取指定主题的异步生产者，如果不存在则创建
func (k *Kafka) AsyncProducer(topic string) *kafka.Writer {
	k.mu.Lock()
	writer, exists := k.producers[topic]
	k.mu.Unlock()
	if exists {
		return writer
	}

	writer = &kafka.Writer{
		Addr:                   kafka.TCP(k.config.Brokers...),
		Topic:                  topic,
		Balancer:               k.config.balancer(),
		Transport:              k.transport,
		AllowAutoTopicCreation: k.config.AllowAutoTopicCreation,
		Async:                  true,
	}

	k.mu.Lock()
	k.producers[topic] = writer
	k.mu.Unlock()

	return writer
}

// Consumer 获取指定主题的消费者，如果不存在则创建
func (k *Kafka) Consumer(topic string) *kafka.Reader {
	k.mu.Lock()
	reader, exists := k.consumers[topic]
	k.mu.Unlock()
	if exists {
		return reader
	}

	reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:   k.config.Brokers,
		Topic:     topic,
		Partition: k.config.Partition,
		MinBytes:  int(k.config.MinBytes),
		MaxBytes:  int(k.config.MaxBytes),
		Dialer:    k.dialer,
	})

	k.mu.Lock()
	k.consumers[topic] = reader
	k.mu.Unlock()

	return reader
}

// ConsumerGroup 获取指定主题和消费者组的消费者，如果不存在则创建
func (k *Kafka) ConsumerGroup(topic string, groupId string) *kafka.Reader {
	var builder strings.Builder
	builder.WriteString(topic)
	builder.WriteString("-")
	builder.WriteString(groupId)
	key := builder.String()

	k.mu.Lock()
	reader, exists := k.consumers[key]
	k.mu.Unlock()
	if exists {
		return reader
	}

	reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:  k.config.Brokers,
		GroupID:  groupId,
		Topic:    topic,
		MinBytes: int(k.config.MinBytes),
		MaxBytes: int(k.config.MaxBytes),
		Dialer:   k.dialer,
	})

	k.mu.Lock()
	k.consumers[key] = reader
	k.mu.Unlock()

	return reader
}

// Close 关闭所有的生产者和消费者连接
func (k *Kafka) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(k.config.CloseTimeout)*time.Second)
	defer cancel()

	eg, _ := errgroup.WithContext(ctx)

	// 并发关闭所有生产者
	for _, producer := range k.producers {
		producer := producer
		eg.Go(func() error {
			return producer.Close()
		})
	}
	// 并发关闭所有消费者
	for _, consumer := range k.consumers {
		consumer := consumer
		eg.Go(func() error {
			return consumer.Close()
		})
	}

	return eg.Wait()
}
