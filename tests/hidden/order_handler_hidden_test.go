package hidden

// Hidden Test 7 (TICKET-2026-0741): verifies sellerId/paymentStatus query
// params actually reach the repository through the handler -> service ->
// repository chain, using a fake repository so this doesn't depend on SQL.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/internal/api/handlers"
	"order-service/internal/api/routes"
	"order-service/internal/domain"
	"order-service/internal/service"
)

type capturingOrderRepo struct {
	lastFilter domain.OrderFilter
}

func (c *capturingOrderRepo) List(ctx context.Context, filter domain.OrderFilter, pagination domain.Pagination) ([]domain.Order, int, error) {
	c.lastFilter = filter
	return nil, 0, nil
}

func (c *capturingOrderRepo) Create(ctx context.Context, req domain.CreateOrderRequest) (*domain.Order, error) {
	return nil, nil
}

func TestHidden_SellerAndPaymentStatusFiltersReachRepository(t *testing.T) {
	repo := &capturingOrderRepo{}
	svc := service.NewOrderService(repo, nil)
	handler := handlers.NewOrderHandler(svc)
	router := routes.NewRouter(handler)

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/orders?sellerId=seller-42&paymentStatus=PAID")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotNil(t, repo.lastFilter.SellerID, "sellerId query param did not reach the repository filter")
	assert.Equal(t, "seller-42", *repo.lastFilter.SellerID)
	require.NotNil(t, repo.lastFilter.PaymentStatus, "paymentStatus query param did not reach the repository filter")
	assert.Equal(t, "PAID", *repo.lastFilter.PaymentStatus)
}
