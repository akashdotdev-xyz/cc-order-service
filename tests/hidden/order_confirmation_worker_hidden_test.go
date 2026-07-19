package hidden

// Hidden tests for Problem 4 (Add Background Worker): the worker must
// unmarshal OrderCreated events and confirm the order, and must tolerate
// being handed the same message more than once (at-least-once delivery).

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"order-service/internal/domain"
	"order-service/internal/queue"
	"order-service/internal/queue/kafka"
	"order-service/internal/worker"
)

type fakeConfirmRepo struct {
	confirmedOrderIDs []string
}

func (f *fakeConfirmRepo) List(ctx context.Context, filter domain.OrderFilter, pagination domain.Pagination) ([]domain.Order, int, error) {
	return nil, 0, nil
}

func (f *fakeConfirmRepo) Create(ctx context.Context, req domain.CreateOrderRequest) (*domain.Order, error) {
	return nil, nil
}

func (f *fakeConfirmRepo) ConfirmOrder(ctx context.Context, orderID string) error {
	f.confirmedOrderIDs = append(f.confirmedOrderIDs, orderID)
	return nil
}

func orderCreatedMessage(t *testing.T, orderID string) kafka.Message {
	t.Helper()
	value, err := json.Marshal(queue.OrderCreatedEvent{
		OrderID:     orderID,
		CustomerID:  "cust-1",
		WarehouseID: "wh-1",
		SellerID:    "seller-1",
		TotalAmount: 1000,
		OccurredAt:  time.Now(),
	})
	require.NoError(t, err)
	return kafka.Message{Key: []byte(orderID), Value: value}
}

func TestHidden_OrderConfirmationWorker_ConfirmsOrderFromEvent(t *testing.T) {
	repo := &fakeConfirmRepo{}
	w := worker.NewOrderConfirmationWorker(repo)

	err := w.HandleMessage(context.Background(), orderCreatedMessage(t, "order-1"))
	require.NoError(t, err)
	require.Equal(t, []string{"order-1"}, repo.confirmedOrderIDs)
}

func TestHidden_OrderConfirmationWorker_MalformedPayloadReturnsError(t *testing.T) {
	repo := &fakeConfirmRepo{}
	w := worker.NewOrderConfirmationWorker(repo)

	err := w.HandleMessage(context.Background(), kafka.Message{Value: []byte("not json")})
	require.Error(t, err)
	require.Empty(t, repo.confirmedOrderIDs, "must not attempt to confirm an order from an unparseable message")
}

func TestHidden_OrderConfirmationWorker_RedeliveredMessageDoesNotError(t *testing.T) {
	repo := &fakeConfirmRepo{}
	w := worker.NewOrderConfirmationWorker(repo)

	msg := orderCreatedMessage(t, "order-1")
	require.NoError(t, w.HandleMessage(context.Background(), msg))
	require.NoError(t, w.HandleMessage(context.Background(), msg),
		"at-least-once delivery means the same message can arrive twice — handling it again must not error")
	require.Equal(t, []string{"order-1", "order-1"}, repo.confirmedOrderIDs)
}
