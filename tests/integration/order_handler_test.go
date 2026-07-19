package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/internal/api/handlers"
	"order-service/internal/api/routes"
	"order-service/internal/domain"
	"order-service/internal/service"
)

type fakeOrderRepo struct {
	orders []domain.Order
}

func (f *fakeOrderRepo) List(ctx context.Context, filter domain.OrderFilter, pagination domain.Pagination) ([]domain.Order, int, error) {
	return f.orders, len(f.orders), nil
}

func (f *fakeOrderRepo) Create(ctx context.Context, req domain.CreateOrderRequest) (*domain.Order, error) {
	return nil, nil
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	repo := &fakeOrderRepo{
		orders: []domain.Order{
			{ID: "1", Status: domain.OrderStatusPending, WarehouseID: "wh-1", CreatedAt: time.Now()},
		},
	}
	svc := service.NewOrderService(repo, nil)
	handler := handlers.NewOrderHandler(svc)
	router := routes.NewRouter(handler)
	return httptest.NewServer(router)
}

func TestListOrders_Success(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/orders")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result domain.PagedResult
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Len(t, result.Orders, 1)
	assert.Equal(t, 1, result.Total)
}

func TestListOrders_InvalidPageReturns400(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/orders?page=-1")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestListOrders_InvalidCreatedAfterReturns400(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/orders?createdAfter=not-a-timestamp")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestListOrders_ValidFiltersReturn200(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	url := srv.URL + "/orders?status=PENDING&warehouseId=wh-1&page=1&limit=10&sortBy=createdAt&sortOrder=asc"
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
