package service

import (
	"context"
	"errors"
	"math"
	"strings"

	"order-service/internal/domain"
	"order-service/internal/repository"
)

var (
	ErrInvalidPage        = errors.New("page must be >= 1")
	ErrInvalidLimit       = errors.New("limit must be between 1 and 100")
	ErrInvalidDateRange   = errors.New("createdAfter must be before createdBefore")
	ErrInvalidOrderStatus = errors.New("invalid order status")
	ErrInvalidSortOrder   = errors.New("sortOrder must be 'asc' or 'desc'")

	ErrMissingCustomerID   = errors.New("customerId is required")
	ErrMissingWarehouseID  = errors.New("warehouseId is required")
	ErrMissingSellerID     = errors.New("sellerId is required")
	ErrEmptyItems          = errors.New("items must not be empty")
	ErrInvalidItemQuantity = errors.New("each item must have a sku and a quantity > 0")
)

const (
	DefaultPage  = 1
	DefaultLimit = 20
	MaxLimit     = 100
)

type OrderService struct {
	repo repository.OrderRepository
}

func NewOrderService(repo repository.OrderRepository) *OrderService {
	return &OrderService{repo: repo}
}

// ListOrders validates and defaults the incoming filter/pagination, then
// delegates to the repository.
func (s *OrderService) ListOrders(ctx context.Context, filter domain.OrderFilter, pagination domain.Pagination) (*domain.PagedResult, error) {
	if pagination.Page == 0 {
		pagination.Page = DefaultPage
	}
	if pagination.Limit == 0 {
		pagination.Limit = DefaultLimit
	}

	if pagination.Page < 1 {
		return nil, ErrInvalidPage
	}
	if pagination.Limit < 1 || pagination.Limit > MaxLimit {
		return nil, ErrInvalidLimit
	}

	if filter.Status != nil && !isValidStatus(*filter.Status) {
		return nil, ErrInvalidOrderStatus
	}

	if filter.CreatedAfter != nil && filter.CreatedBefore != nil {
		if !filter.CreatedAfter.Before(*filter.CreatedBefore) {
			return nil, ErrInvalidDateRange
		}
	}

	if pagination.SortOrder != "" {
		order := strings.ToLower(pagination.SortOrder)
		if order != "asc" && order != "desc" {
			return nil, ErrInvalidSortOrder
		}
	}

	orders, total, err := s.repo.List(ctx, filter, pagination)
	if err != nil {
		return nil, err
	}

	return &domain.PagedResult{
		Orders:     orders,
		Page:       pagination.Page,
		Limit:      pagination.Limit,
		Total:      total,
		TotalPages: int(math.Ceil(float64(total) / float64(pagination.Limit))),
	}, nil
}

// CreateOrder validates the request and delegates to the repository to
// reserve inventory and persist the order.
//
// TODO(candidate): implement this method.
//
// Requirements:
//   - Return ErrMissingCustomerID / ErrMissingWarehouseID / ErrMissingSellerID
//     if the corresponding field is empty.
//   - Return ErrEmptyItems if req.Items is empty.
//   - Return ErrInvalidItemQuantity if any item has an empty SKU or a
//     Quantity <= 0.
//   - On success, call s.repo.Create and return its result as-is —
//     including propagating domain.ErrInsufficientInventory unchanged, so
//     the handler can map it to its own status code without the service
//     needing to know about HTTP.
func (s *OrderService) CreateOrder(ctx context.Context, req domain.CreateOrderRequest) (*domain.Order, error) {

	validationErr := validateCreateOrderRequest(req)
	if validationErr != nil {
		return nil, validationErr
	}

	return s.repo.Create(ctx, req)
}

func isValidStatus(s domain.OrderStatus) bool {
	switch s {
	case domain.OrderStatusPending, domain.OrderStatusConfirmed, domain.OrderStatusShipped,
		domain.OrderStatusDelivered, domain.OrderStatusCancelled:
		return true
	default:
		return false
	}
}

func validateCreateOrderRequest(req domain.CreateOrderRequest) error {
	if req.CustomerID == "" {
		return ErrMissingCustomerID
	}

	if req.SellerID == "" {
		return ErrMissingSellerID
	}

	if req.WarehouseID == "" {
		return ErrMissingWarehouseID
	}

	if len(req.Items) == 0 {
		return ErrEmptyItems
	}

	for _, item := range req.Items {
		if item.SKU == "" || item.Quantity <= 0 {
			return ErrInvalidItemQuantity
		}
	}
	return nil
}
