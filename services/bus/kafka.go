package bus

import (
	"context"
	"fmt"
	"strings"

	"github.com/segmentio/kafka-go"
	"ponctuel/services/events"
	"ponctuel/services/ingester/domain"
)

type Publisher interface {
	Publish(context.Context, string, string, domain.Event) error
	Close() error
}

type KafkaPublisher struct {
	writers map[string]*kafka.Writer
}

func NewKafkaPublisher(brokers []string) (*KafkaPublisher, error) {
	clean := make([]string, 0, len(brokers))
	for _, broker := range brokers {
		if value := strings.TrimSpace(broker); value != "" {
			clean = append(clean, value)
		}
	}
	if len(clean) == 0 {
		return nil, fmt.Errorf("at least one Redpanda broker is required")
	}
	return &KafkaPublisher{writers: map[string]*kafka.Writer{
		events.TopicTripUpdates:      newWriter(clean, events.TopicTripUpdates),
		events.TopicVehiclePositions: newWriter(clean, events.TopicVehiclePositions),
	}}, nil
}

func (p *KafkaPublisher) Publish(ctx context.Context, topic, key string, event domain.Event) error {
	if p == nil {
		return fmt.Errorf("Kafka publisher is nil")
	}
	writer, ok := p.writers[topic]
	if !ok {
		return fmt.Errorf("unsupported Redpanda topic %q", topic)
	}
	payload, err := events.Encode(event)
	if err != nil {
		return fmt.Errorf("encode event: %w", err)
	}
	if err := writer.WriteMessages(ctx, kafka.Message{Key: []byte(key), Value: payload}); err != nil {
		return fmt.Errorf("publish %s: %w", topic, err)
	}
	return nil
}

func (p *KafkaPublisher) Close() error {
	if p == nil {
		return nil
	}
	var firstErr error
	for _, writer := range p.writers {
		if err := writer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func newWriter(brokers []string, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}
}

func messageFor(event domain.Event, key string) (kafka.Message, error) {
	payload, err := events.Encode(event)
	if err != nil {
		return kafka.Message{}, err
	}
	return kafka.Message{Key: []byte(key), Value: payload}, nil
}
