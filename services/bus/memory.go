package bus

import (
	"context"
	"fmt"
	"sync"

	"github.com/segmentio/kafka-go"
	"ponctuel/services/events"
	"ponctuel/services/ingester/domain"
)

const defaultMemoryBusBuffer = 256

// MemoryBus provides the same Publisher/Reader contract as Kafka for the
// single-container public demo. It keeps the production event contract while
// avoiding a broker that cannot be persisted on Render Free.
type MemoryBus struct {
	channels map[string]chan kafka.Message
	closed   chan struct{}
	once     sync.Once
}

func NewMemoryBus(buffer int) *MemoryBus {
	if buffer <= 0 {
		buffer = defaultMemoryBusBuffer
	}
	return &MemoryBus{
		channels: map[string]chan kafka.Message{
			events.TopicTripUpdates:      make(chan kafka.Message, buffer),
			events.TopicVehiclePositions: make(chan kafka.Message, buffer),
		},
		closed: make(chan struct{}),
	}
}

func (b *MemoryBus) Publish(ctx context.Context, topic, key string, event domain.Event) error {
	if b == nil {
		return fmt.Errorf("memory bus is nil")
	}
	payload, err := events.Encode(event)
	if err != nil {
		return fmt.Errorf("encode event: %w", err)
	}
	channel, ok := b.channels[topic]
	if !ok {
		return fmt.Errorf("unsupported memory bus topic %q", topic)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-b.closed:
		return fmt.Errorf("memory bus is closed")
	case <-ctx.Done():
		return ctx.Err()
	case channel <- kafka.Message{Topic: topic, Key: []byte(key), Value: payload}:
		return nil
	}
}

func (b *MemoryBus) Readers() []Reader {
	if b == nil {
		return nil
	}
	return []Reader{
		&memoryReader{bus: b, topic: events.TopicTripUpdates},
		&memoryReader{bus: b, topic: events.TopicVehiclePositions},
	}
}

func (b *MemoryBus) Close() error {
	if b != nil {
		b.once.Do(func() { close(b.closed) })
	}
	return nil
}

type memoryReader struct {
	bus   *MemoryBus
	topic string
}

func (r *memoryReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	if r == nil || r.bus == nil {
		return kafka.Message{}, fmt.Errorf("memory reader is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-r.bus.closed:
		return kafka.Message{}, fmt.Errorf("memory bus is closed")
	case <-ctx.Done():
		return kafka.Message{}, ctx.Err()
	case message := <-r.bus.channels[r.topic]:
		return message, nil
	}
}

func (*memoryReader) CommitMessages(context.Context, ...kafka.Message) error { return nil }

func (*memoryReader) Close() error { return nil }
