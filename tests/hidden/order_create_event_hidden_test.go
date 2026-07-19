package hidden

// Hidden tests for Problem 3 (Add Kafka Events): a successful CreateOrder
// must publish an OrderCreated event; a failed one must not.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"order-service/internal/domain"
	"order-service/internal/queue"
	"order-service/internal/service"
)

type fakeCreateOnlyRepo struct {
	order *domain.Order
	err   error
}

func (f *fakeCreateOnlyRepo) List(ctx context.Context, filter domain.OrderFilter, pagination domain.Pagination) ([]domain.Order, int, error) {
	return nil, 0, nil
}

func (f *fakeCreateOnlyRepo) Create(ctx context.Context, req domain.CreateOrderRequest) (*domain.Order, error) {
	return f.order, f.err
}

type fakeEventPublisher struct {
	events []queue.OrderCreatedEvent
}

func (f *fakeEventPublisher) PublishOrderCreated(ctx context.Context, event queue.OrderCreatedEvent) error {
	f.events = append(f.events, event)
	return nil
}

func validCreateOrderRequest() domain.CreateOrderRequest {
	return domain.CreateOrderRequest{
		CustomerID:  "cust-1",
		WarehouseID: "wh-1",
		SellerID:    "seller-1",
		TotalAmount: 4200,
		Items:       []domain.OrderItem{{SKU: "sku-a", Quantity: 2}},
	}
}

func TestHidden_CreateOrder_PublishesEventOnSuccess(t *testing.T) {
	created := &domain.Order{
		ID:          "order-1",
		CustomerID:  "cust-1",
		WarehouseID: "wh-1",
		SellerID:    "seller-1",
		TotalAmount: 4200,
		Items:       []domain.OrderItem{{SKU: "sku-a", Quantity: 2}},
	}
	repo := &fakeCreateOnlyRepo{order: created}
	pub := &fakeEventPublisher{}
	svc := service.NewOrderService(repo, pub)

	_, err := svc.CreateOrder(context.Background(), validCreateOrderRequest())
	require.NoError(t, err)
	require.Len(t, pub.events, 1, "expected exactly one OrderCreated event to be published")
	require.Equal(t, "order-1", pub.events[0].OrderID)
}

func TestHidden_CreateOrder_DoesNotPublishOnRepositoryFailure(t *testing.T) {
	repo := &fakeCreateOnlyRepo{err: domain.ErrInsufficientInventory}
	pub := &fakeEventPublisher{}
	svc := service.NewOrderService(repo, pub)

	_, err := svc.CreateOrder(context.Background(), validCreateOrderRequest())
	require.ErrorIs(t, err, domain.ErrInsufficientInventory)
	require.Empty(t, pub.events, "must not publish an event for an order that was never created")
}
