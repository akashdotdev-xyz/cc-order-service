package unit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/internal/domain"
	"order-service/internal/service"
)

// fakeOrderRepo is an in-memory stand-in for repository.OrderRepository used
// to unit-test the service layer in isolation from SQL.
type fakeOrderRepo struct {
	orders   []domain.Order
	lastCall struct {
		filter     domain.OrderFilter
		pagination domain.Pagination
	}
}

func (f *fakeOrderRepo) List(ctx context.Context, filter domain.OrderFilter, pagination domain.Pagination) ([]domain.Order, int, error) {
	f.lastCall.filter = filter
	f.lastCall.pagination = pagination
	return f.orders, len(f.orders), nil
}

func TestListOrders_DefaultsPageAndLimit(t *testing.T) {
	repo := &fakeOrderRepo{orders: []domain.Order{{ID: "1"}}}
	svc := service.NewOrderService(repo)

	result, err := svc.ListOrders(context.Background(), domain.OrderFilter{}, domain.Pagination{})
	require.NoError(t, err)

	assert.Equal(t, service.DefaultPage, repo.lastCall.pagination.Page)
	assert.Equal(t, service.DefaultLimit, repo.lastCall.pagination.Limit)
	assert.Equal(t, 1, result.Total)
}

func TestListOrders_InvalidPage(t *testing.T) {
	repo := &fakeOrderRepo{}
	svc := service.NewOrderService(repo)

	_, err := svc.ListOrders(context.Background(), domain.OrderFilter{}, domain.Pagination{Page: 0})
	// Page: 0 should be treated as "unset" and defaulted, not an error.
	assert.NoError(t, err)

	_, err = svc.ListOrders(context.Background(), domain.OrderFilter{}, domain.Pagination{Page: -1})
	assert.ErrorIs(t, err, service.ErrInvalidPage)
}

func TestListOrders_InvalidLimit(t *testing.T) {
	repo := &fakeOrderRepo{}
	svc := service.NewOrderService(repo)

	_, err := svc.ListOrders(context.Background(), domain.OrderFilter{}, domain.Pagination{Limit: 1000})
	assert.ErrorIs(t, err, service.ErrInvalidLimit)

	_, err = svc.ListOrders(context.Background(), domain.OrderFilter{}, domain.Pagination{Limit: -5})
	assert.ErrorIs(t, err, service.ErrInvalidLimit)
}

func TestListOrders_InvalidStatus(t *testing.T) {
	repo := &fakeOrderRepo{}
	svc := service.NewOrderService(repo)

	badStatus := domain.OrderStatus("NOT_A_REAL_STATUS")
	_, err := svc.ListOrders(context.Background(), domain.OrderFilter{Status: &badStatus}, domain.Pagination{})
	assert.ErrorIs(t, err, service.ErrInvalidOrderStatus)
}

func TestListOrders_InvalidDateRange(t *testing.T) {
	repo := &fakeOrderRepo{}
	svc := service.NewOrderService(repo)

	after := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	before := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err := svc.ListOrders(context.Background(), domain.OrderFilter{
		CreatedAfter:  &after,
		CreatedBefore: &before,
	}, domain.Pagination{})
	assert.ErrorIs(t, err, service.ErrInvalidDateRange)
}

func TestListOrders_InvalidSortOrder(t *testing.T) {
	repo := &fakeOrderRepo{}
	svc := service.NewOrderService(repo)

	_, err := svc.ListOrders(context.Background(), domain.OrderFilter{}, domain.Pagination{SortOrder: "sideways"})
	assert.ErrorIs(t, err, service.ErrInvalidSortOrder)
}

func TestListOrders_ComputesTotalPages(t *testing.T) {
	orders := make([]domain.Order, 25)
	repo := &fakeOrderRepo{orders: orders}
	svc := service.NewOrderService(repo)

	result, err := svc.ListOrders(context.Background(), domain.OrderFilter{}, domain.Pagination{Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 25, result.Total)
	assert.Equal(t, 3, result.TotalPages)
}
