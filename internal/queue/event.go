package queue

import (
	"context"
	"time"

	"order-service/internal/domain"
)

// OrderCreatedEvent is the wire schema published whenever an order is
// successfully created. Kept separate from domain.Order so the event
// contract can evolve independently of the internal model.
type OrderCreatedEvent struct {
	OrderID     string             `json:"orderId"`
	CustomerID  string             `json:"customerId"`
	WarehouseID string             `json:"warehouseId"`
	SellerID    string             `json:"sellerId"`
	TotalAmount int64              `json:"totalAmount"`
	Items       []domain.OrderItem `json:"items"`
	OccurredAt  time.Time          `json:"occurredAt"`
}

// Publisher publishes domain events for the order service. Implementations
// must be safe for concurrent use.
type Publisher interface {
	PublishOrderCreated(ctx context.Context, event OrderCreatedEvent) error
}
