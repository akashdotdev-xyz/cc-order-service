package service

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"order-service/internal/domain"
	"order-service/internal/queue"
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
	repo      repository.OrderRepository
	publisher queue.Publisher
}

func NewOrderService(repo repository.OrderRepository, publisher queue.Publisher) *OrderService {
	return &OrderService{repo: repo, publisher: publisher}
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

// CreateOrder validates the request, delegates to the repository to
// reserve inventory and persist the order, then publishes an OrderCreated
// event.
//
// TODO(candidate): publish the OrderCreated event after a successful create.
//
// Requirements:
//   - Only publish once s.repo.Create has actually succeeded — never
//     publish an event for an order that was never persisted.
//   - Build a queue.OrderCreatedEvent from the created *domain.Order (see
//     internal/queue/event.go for its fields) and call
//     s.publisher.PublishOrderCreated.
//   - A publish failure must NOT cause CreateOrder to return an error — the
//     order is already durably committed at this point. Log the failure
//     instead of propagating it (this gap — a lost event when publish
//     fails after commit — is intentional for now and gets closed later by
//     an outbox pattern).
//   - s.publisher may be nil (some existing callers don't care about
//     events) — guard against that before calling it.
func (s *OrderService) CreateOrder(ctx context.Context, req domain.CreateOrderRequest) (*domain.Order, error) {

	validationErr := validateCreateOrderRequest(req)
	if validationErr != nil {
		return nil, validationErr
	}

	order, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	if s.publisher != nil {
		event := queue.OrderCreatedEvent{
			OrderID:     order.ID,
			CustomerID:  order.CustomerID,
			WarehouseID: order.WarehouseID,
			SellerID:    order.SellerID,
			TotalAmount: order.TotalAmount,
			Items:       order.Items,
			OccurredAt:  time.Now(),
		}
		s.publisher.PublishOrderCreated(ctx, event)
	}
	return order, nil
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
