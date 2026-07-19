package hidden

// Hidden tests for Problem 2 (Integrate Inventory): POST /orders must
// reserve inventory and create the order atomically, with row-level locking
// so concurrent requests can't both reserve more stock than is available.

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"order-service/internal/domain"
)

func TestHidden_CreateOrder_LocksAndReservesInventory(t *testing.T) {
	repo, mock, closeFn := newMockRepo(t)
	defer closeFn()

	req := domain.CreateOrderRequest{
		CustomerID:  "cust-1",
		WarehouseID: "wh-1",
		SellerID:    "seller-1",
		TotalAmount: 5000,
		Items:       []domain.OrderItem{{SKU: "sku-a", Quantity: 3}},
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`(?i)SELECT quantity_available FROM inventory WHERE.*FOR UPDATE`).
		WithArgs(req.WarehouseID, "sku-a").
		WillReturnRows(sqlmock.NewRows([]string{"quantity_available"}).AddRow(10))
	mock.ExpectExec(`(?i)UPDATE inventory SET quantity_available`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?i)INSERT INTO orders`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "version", "created_at", "updated_at"}).
			AddRow("order-1", 1, time.Now(), time.Now()))
	mock.ExpectExec(`(?i)INSERT INTO order_items`).
		WithArgs(sqlmock.AnyArg(), "sku-a", 3).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	order, err := repo.Create(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, order)
	require.Equal(t, "order-1", order.ID)
	require.Len(t, order.Items, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHidden_CreateOrder_InsufficientInventoryRollsBack(t *testing.T) {
	repo, mock, closeFn := newMockRepo(t)
	defer closeFn()

	req := domain.CreateOrderRequest{
		CustomerID:  "cust-1",
		WarehouseID: "wh-1",
		SellerID:    "seller-1",
		TotalAmount: 5000,
		Items:       []domain.OrderItem{{SKU: "sku-a", Quantity: 999}},
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`(?i)SELECT quantity_available FROM inventory WHERE.*FOR UPDATE`).
		WithArgs(req.WarehouseID, "sku-a").
		WillReturnRows(sqlmock.NewRows([]string{"quantity_available"}).AddRow(10))
	mock.ExpectRollback()

	_, err := repo.Create(context.Background(), req)
	require.ErrorIs(t, err, domain.ErrInsufficientInventory)
	require.NoError(t, mock.ExpectationsWereMet(),
		"on insufficient inventory the transaction must roll back with no order/order_items rows inserted")
}

func TestHidden_CreateOrder_MultiItemReservesEachRow(t *testing.T) {
	repo, mock, closeFn := newMockRepo(t)
	defer closeFn()

	req := domain.CreateOrderRequest{
		CustomerID:  "cust-1",
		WarehouseID: "wh-1",
		SellerID:    "seller-1",
		TotalAmount: 8000,
		Items: []domain.OrderItem{
			{SKU: "sku-a", Quantity: 2},
			{SKU: "sku-b", Quantity: 1},
		},
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`(?i)SELECT quantity_available FROM inventory WHERE.*FOR UPDATE`).
		WithArgs(req.WarehouseID, "sku-a").
		WillReturnRows(sqlmock.NewRows([]string{"quantity_available"}).AddRow(5))
	mock.ExpectExec(`(?i)UPDATE inventory SET quantity_available`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?i)SELECT quantity_available FROM inventory WHERE.*FOR UPDATE`).
		WithArgs(req.WarehouseID, "sku-b").
		WillReturnRows(sqlmock.NewRows([]string{"quantity_available"}).AddRow(3))
	mock.ExpectExec(`(?i)UPDATE inventory SET quantity_available`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?i)INSERT INTO orders`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "version", "created_at", "updated_at"}).
			AddRow("order-2", 1, time.Now(), time.Now()))
	mock.ExpectExec(`(?i)INSERT INTO order_items`).
		WithArgs(sqlmock.AnyArg(), "sku-a", 2).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?i)INSERT INTO order_items`).
		WithArgs(sqlmock.AnyArg(), "sku-b", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	order, err := repo.Create(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, order.Items, 2)
	require.NoError(t, mock.ExpectationsWereMet())
}
