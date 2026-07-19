package domain

import "errors"

// ErrInsufficientInventory is returned when an order cannot be fully
// reserved against available stock. Repository implementations must roll
// back any partial reservation before returning it.
var ErrInsufficientInventory = errors.New("insufficient inventory")

// OrderItem is a single line item within an order.
type OrderItem struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

// Inventory tracks available stock for a SKU within a warehouse.
type Inventory struct {
	WarehouseID       string `json:"warehouseId"`
	SKU               string `json:"sku"`
	QuantityAvailable int    `json:"quantityAvailable"`
	Version           int    `json:"version"`
}

// CreateOrderRequest is the payload for POST /orders. TotalAmount is
// caller-supplied (this service does not own pricing).
type CreateOrderRequest struct {
	CustomerID  string      `json:"customerId"`
	WarehouseID string      `json:"warehouseId"`
	SellerID    string      `json:"sellerId"`
	TotalAmount int64       `json:"totalAmount"`
	Items       []OrderItem `json:"items"`
}
