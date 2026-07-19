package worker

import (
	"context"
	"encoding/json"

	"order-service/internal/queue"
	"order-service/internal/queue/kafka"
	"order-service/internal/repository"
)

// OrderConfirmationWorker consumes OrderCreated events and transitions the
// corresponding order from PENDING to CONFIRMED.
type OrderConfirmationWorker struct {
	repo repository.OrderRepository
}

func NewOrderConfirmationWorker(repo repository.OrderRepository) *OrderConfirmationWorker {
	return &OrderConfirmationWorker{repo: repo}
}

// HandleMessage processes a single Kafka message carrying a
// queue.OrderCreatedEvent.
//
// TODO(candidate): implement this method.
//
// Requirements:
//   - Unmarshal msg.Value into a queue.OrderCreatedEvent. On a malformed
//     payload, return the unmarshal error.
//   - Call w.repo.ConfirmOrder(ctx, event.OrderID) and return its error.
//   - This method WILL be called more than once for the same message under
//     Kafka's at-least-once delivery (consumer restarts, rebalances,
//     redelivery after a slow ack, etc.) — it must be safe to call
//     repeatedly without erroring or double-applying anything. You don't
//     need extra dedup logic here: ConfirmOrder is already idempotent on
//     its own, so just don't add anything (like treating "already
//     confirmed" as an error) that breaks that.
func (w *OrderConfirmationWorker) HandleMessage(ctx context.Context, msg kafka.Message) error {
	// TODO: unmarshal msg.Value into a queue.OrderCreatedEvent and call
	// w.repo.ConfirmOrder.

	var event queue.OrderCreatedEvent
	err := json.Unmarshal(msg.Value, &event)
	if err != nil {
		return err
	}

	return w.repo.ConfirmOrder(ctx, event.OrderID)
}
