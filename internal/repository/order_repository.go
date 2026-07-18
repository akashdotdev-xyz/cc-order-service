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
}
