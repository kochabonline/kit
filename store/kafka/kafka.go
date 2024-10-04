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

type Kafka struct {
	config    *Config
	dialer    *kafka.Dialer
	transport *kafka.Transport
	producers map[string]*kafka.Writer
	consumers map[string]*kafka.Reader
	mu        sync.Mutex
}

type Option func(*Kafka)

func New(c *Config, opts ...Option) (*Kafka, error) {
	k := &Kafka{
		config:    c,
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

func (k *Kafka) mechanism() plain.Mechanism {
	return plain.Mechanism{
		Username: k.config.Username,
		Password: k.config.Password,
	}
}

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

func (k *Kafka) createTransport() *kafka.Transport {
	transport := &kafka.Transport{}

	if k.config.Username != "" && k.config.Password != "" {
		transport.SASL = k.mechanism()
	}

	return transport
}

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

func (k *Kafka) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(k.config.CloseTimeout)*time.Second)
	defer cancel()

	eg, _ := errgroup.WithContext(ctx)

	for _, producer := range k.producers {
		producer := producer
		eg.Go(func() error {
			return producer.Close()
		})
	}
	for _, consumer := range k.consumers {
		consumer := consumer
		eg.Go(func() error {
			return consumer.Close()
		})
	}

	return eg.Wait()
}
