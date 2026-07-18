package domain

import (
	"errors"
	"time"
)

// OrderStatus represents the lifecycle state of an order.
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusConfirmed OrderStatus = "CONFIRMED"
	OrderStatusShipped   OrderStatus = "SHIPPED"
	OrderStatusDelivered OrderStatus = "DELIVERED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
)

var validStatus = map[string]OrderStatus{
	"PENDING":   OrderStatusPending,
	"CONFIRMED": OrderStatusConfirmed,
	"SHIPPED":   OrderStatusShipped,
	"DELIVERED": OrderStatusDelivered,
	"CANCELLED": OrderStatusCancelled,
}

func ParseOrderStatus(s string) (*OrderStatus, error) {
	if s == "" {
		return nil, nil
	}

	status, ok := validStatus[s]
	if !ok {
		return nil, errors.New("not a valid status")
	}
	return &status, nil
}

// Order is the core domain entity for this service.
type Order struct {
	ID            string      `json:"id"`
	CustomerID    string      `json:"customerId"`
	WarehouseID   string      `json:"warehouseId"`
	SellerID      string      `json:"sellerId"`
	Status        OrderStatus `json:"status"`
	PaymentStatus string      `json:"paymentStatus"`
	TotalAmount   int64       `json:"totalAmount"` // stored in minor units (cents)
	Version       int         `json:"version"`     // used for optimistic locking in later problems
	CreatedAt     time.Time   `json:"createdAt"`
	UpdatedAt     time.Time   `json:"updatedAt"`
}

// OrderFilter captures the supported query filters for listing orders.
//
// Pointer fields are optional; a nil pointer means "no filter applied".
type OrderFilter struct {
	Status        *OrderStatus
	WarehouseID   *string
	SellerID      *string
	PaymentStatus *string
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
}

// Pagination captures paging + sorting parameters for a list endpoint.
type Pagination struct {
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

// PagedResult wraps a page of orders with pagination metadata.
type PagedResult struct {
	Orders     []Order `json:"orders"`
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
	Total      int     `json:"total"`
	TotalPages int     `json:"totalPages"`
}
