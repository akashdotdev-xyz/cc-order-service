package kafka

import (
	"context"
	"encoding/json"

	"order-service/internal/queue"
)

// Message is the minimal Kafka message shape this package depends on —
// decoupled from any specific client library so production and test
// writers can both satisfy Writer without this package needing a real
// Kafka client as a dependency.
type Message struct {
	Key   []byte
	Value []byte
}

// Writer is the subset of a Kafka producer client this package needs.
// Satisfied by a real client's writer in production, and by a fake in
// tests.
type Writer interface {
	WriteMessages(ctx context.Context, msgs ...Message) error
}

// Publisher publishes order-service domain events to a single Kafka topic.
type Publisher struct {
	writer Writer
	topic  string
}

func NewPublisher(writer Writer, topic string) *Publisher {
	return &Publisher{writer: writer, topic: topic}
}

// PublishOrderCreated publishes an OrderCreated event.
//
// TODO(candidate): implement this method.
//
// Requirements:
//   - Marshal event to JSON as the message value.
//   - Key the message with event.OrderID (as []byte). Keying by order ID
//     means every event for a given order lands on the same partition,
//     which is what preserves per-order ordering once more event types are
//     added later.
//   - Call p.writer.WriteMessages with a single Message.
//   - Wrap and return any marshal/write error.
func (p *Publisher) PublishOrderCreated(ctx context.Context, event queue.OrderCreatedEvent) error {
	valueBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := Message{
		Key:   []byte(event.OrderID),
		Value: valueBytes,
	}
	return p.writer.WriteMessages(ctx, msg)
}
