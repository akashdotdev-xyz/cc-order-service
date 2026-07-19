package repository

import (
	"context"

	"order-service/internal/domain"
)

// OrderRepository is the contract the service layer depends on.
//
// DO NOT change this interface — it is a public contract used by the
// service layer and by the hidden test suite.
type OrderRepository interface {
	// List returns a page of orders matching the given filter, along with
	// the total count of matching rows (ignoring pagination) for building
	// pagination metadata.
	List(ctx context.Context, filter domain.OrderFilter, pagination domain.Pagination) ([]domain.Order, int, error)

	// Create reserves inventory for every item in the request and persists
	// the order and its items atomically. If any item cannot be fully
	// reserved, implementations must roll back any reservation already made
	// in this call and return domain.ErrInsufficientInventory.
	Create(ctx context.Context, req domain.CreateOrderRequest) (*domain.Order, error)

	// ConfirmOrder transitions an order from PENDING to CONFIRMED. It must
	// be idempotent: calling it more than once for the same order (e.g.
	// because of at-least-once event delivery) must not error or apply the
	// transition twice. If the order is not currently PENDING (already
	// confirmed, cancelled, etc.), this is a silent no-op.
	ConfirmOrder(ctx context.Context, orderID string) error
}
