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
