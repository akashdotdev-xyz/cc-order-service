package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"order-service/internal/domain"
)

// allowedSortColumns whitelists the columns that may be used in ORDER BY.
// A user-supplied sortBy value is only ever used as a map *lookup key* here
// — the resulting column name always comes from this map, never from the
// request, which is what makes it safe against injection.
var allowedSortColumns = map[string]string{
	"createdAt":   "created_at",
	"totalAmount": "total_amount",
	"status":      "status",
}

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) List(ctx context.Context, filter domain.OrderFilter, pagination domain.Pagination) ([]domain.Order, int, error) {
	conditions, args := buildConditions(filter)

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	total, err := r.count(ctx, where, args)
	if err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	orders, err := r.list(ctx, where, args, pagination)
	if err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}

	return orders, total, nil
}

// buildConditions turns a filter into a slice of parameterized WHERE
// fragments ("status = $1") and their matching argument values, in the same
// order. Every value flows through as a placeholder argument — none of them
// are ever interpolated into the SQL string itself.
func buildConditions(filter domain.OrderFilter) ([]string, []interface{}) {
	var conditions []string
	var args []interface{}

	if filter.Status != nil {
		args = append(args, string(*filter.Status))
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if filter.WarehouseID != nil {
		args = append(args, *filter.WarehouseID)
		conditions = append(conditions, fmt.Sprintf("warehouse_id = $%d", len(args)))
	}
	if filter.SellerID != nil {
		args = append(args, *filter.SellerID)
		conditions = append(conditions, fmt.Sprintf("seller_id = $%d", len(args)))
	}
	if filter.PaymentStatus != nil {
		args = append(args, *filter.PaymentStatus)
		conditions = append(conditions, fmt.Sprintf("payment_status = $%d", len(args)))
	}
	if filter.CreatedAfter != nil {
		args = append(args, *filter.CreatedAfter)
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if filter.CreatedBefore != nil {
		args = append(args, *filter.CreatedBefore)
		conditions = append(conditions, fmt.Sprintf("created_at < $%d", len(args)))
	}

	return conditions, args
}

func (r *OrderRepository) count(ctx context.Context, where string, args []interface{}) (int, error) {
	query := "SELECT COUNT(*) FROM orders" + where

	var total int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *OrderRepository) list(ctx context.Context, where string, args []interface{}, pagination domain.Pagination) ([]domain.Order, error) {
	sortColumn, ok := allowedSortColumns[pagination.SortBy]
	if !ok {
		sortColumn = "created_at"
	}

	sortDirection := "DESC"
	if strings.EqualFold(pagination.SortOrder, "asc") {
		sortDirection = "ASC"
	}

	// Reuse the WHERE args, then append LIMIT/OFFSET as the next two
	// placeholders so their numbering stays consistent with `where`.
	limitArg := len(args) + 1
	offsetArg := len(args) + 2
	listArgs := append(append([]interface{}{}, args...), pagination.Limit, (pagination.Page-1)*pagination.Limit)

	query := fmt.Sprintf(
		`SELECT id, customer_id, warehouse_id, seller_id, status, payment_status, total_amount, version, created_at, updated_at
		 FROM orders %s
		 ORDER BY %s %s, %s %s
		 LIMIT $%d OFFSET $%d`,
		where, sortColumn, sortDirection, "id", sortDirection, limitArg, offsetArg,
	)

	rows, err := r.db.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(
			&o.ID, &o.CustomerID, &o.WarehouseID, &o.SellerID, &o.Status,
			&o.PaymentStatus, &o.TotalAmount, &o.Version, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// Create reserves inventory for each item and inserts the order + its
// items, all within a single transaction.
//
// TODO(candidate): implement this method.
//
// Requirements:
//   - Begin a transaction with r.db.BeginTx(ctx, nil). Every statement below
//     must run inside it, and any error path must roll back before
//     returning.
//   - For each item in req.Items, lock its inventory row with:
//     SELECT quantity_available FROM inventory
//     WHERE warehouse_id = $1 AND sku = $2 FOR UPDATE
//     The FOR UPDATE matters: without it, two concurrent requests can both
//     read the same quantity_available before either writes, and both
//     succeed even though there isn't enough stock for both combined.
//   - If quantity_available < item.Quantity for any item, roll back the
//     transaction and return domain.ErrInsufficientInventory — do not
//     insert anything, and do not leave any earlier item's reservation
//     applied.
//   - Otherwise, decrement that row:
//     UPDATE inventory SET quantity_available = quantity_available - $1
//     WHERE warehouse_id = $2 AND sku = $3
//   - Insert the order (status PENDING, paymentStatus "UNPAID") using
//     RETURNING to get the generated id/version/created_at/updated_at, then
//     insert one row per item into order_items.
//   - Commit. Return the fully populated *domain.Order, including Items.
func (r *OrderRepository) Create(ctx context.Context, req domain.CreateOrderRequest) (*domain.Order, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	for _, item := range req.Items {
		selectQuery := `SELECT quantity_available FROM inventory
			WHERE warehouse_id = $1 AND sku = $2 FOR UPDATE`

		var qtyAvailable int
		if err := tx.QueryRowContext(ctx, selectQuery, req.WarehouseID, item.SKU).Scan(&qtyAvailable); err != nil {
			return nil, err
		}
		if qtyAvailable < item.Quantity {
			return nil, domain.ErrInsufficientInventory
		}

		updateQuery := `UPDATE inventory SET quantity_available = quantity_available - $1
			WHERE warehouse_id = $2 AND sku = $3`
		if _, err := tx.ExecContext(ctx, updateQuery, item.Quantity, req.WarehouseID, item.SKU); err != nil {
			return nil, err
		}
	}

	order := &domain.Order{
		CustomerID:    req.CustomerID,
		WarehouseID:   req.WarehouseID,
		SellerID:      req.SellerID,
		Status:        domain.OrderStatusPending,
		PaymentStatus: "UNPAID",
		TotalAmount:   req.TotalAmount,
	}

	insertOrderQuery := `INSERT INTO orders (customer_id, warehouse_id, seller_id, status, payment_status, total_amount)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, version, created_at, updated_at`

	if err := tx.QueryRowContext(ctx, insertOrderQuery,
		req.CustomerID, req.WarehouseID, req.SellerID, order.Status, order.PaymentStatus, req.TotalAmount,
	).Scan(&order.ID, &order.Version, &order.CreatedAt, &order.UpdatedAt); err != nil {
		return nil, err
	}

	insertItemQuery := `INSERT INTO order_items (order_id, sku, quantity) VALUES ($1, $2, $3)`
	for _, item := range req.Items {
		if _, err := tx.ExecContext(ctx, insertItemQuery, order.ID, item.SKU, item.Quantity); err != nil {
			return nil, err
		}
		order.Items = append(order.Items, item)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return order, nil
}
