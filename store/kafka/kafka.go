package kafka

import (
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
)

type Kafka struct {
	config *Config
}

type Option func(*Kafka)

func New(c *Config, opts ...Option) (*Kafka, error) {
	k := &Kafka{
		config: c,
	}

	if err := k.config.init(); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(k)
	}

	return k, nil
}

func (k *Kafka) dialer() *kafka.Dialer {
	dialer := &kafka.Dialer{
		Timeout:   time.Duration(k.config.Timeout) * time.Second,
		DualStack: true,
	}

	if k.config.Username != "" && k.config.Password != "" {
		mechanism := plain.Mechanism{
			Username: k.config.Username,
			Password: k.config.Password,
		}

		dialer.SASLMechanism = mechanism
	}

	return dialer
}

func (k *Kafka) transport() *kafka.Transport {
	transport := &kafka.Transport{}

	if k.config.Username != "" && k.config.Password != "" {
		mechanism := plain.Mechanism{
			Username: k.config.Username,
			Password: k.config.Password,
		}

		transport.SASL = mechanism
	}

	return transport
}

func (k *Kafka) Producer(topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(k.config.Brokers...),
		Topic:                  topic,
		Balancer:               k.config.balancer(),
		Transport:              k.transport(),
		AllowAutoTopicCreation: k.config.AllowAutoTopicCreation,
	}
}

func (k *Kafka) AsyncProducer(topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(k.config.Brokers...),
		Topic:                  topic,
		Balancer:               k.config.balancer(),
		Transport:              k.transport(),
		AllowAutoTopicCreation: k.config.AllowAutoTopicCreation,
		Async:                  true,
	}
}

func (k *Kafka) Consumer(topic string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:   k.config.Brokers,
		Topic:     topic,
		Partition: k.config.Partition,
		MinBytes:  k.config.MinBytes,
		MaxBytes:  k.config.MaxBytes,
		Dialer:    k.dialer(),
	})
}

func (k *Kafka) ConsumerGroup(topic string, group string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:   k.config.Brokers,
		GroupID:   group,
		Topic:     topic,
		Partition: k.config.Partition,
		MinBytes:  k.config.MinBytes,
		MaxBytes:  k.config.MaxBytes,
		Dialer:    k.dialer(),
	})
}
