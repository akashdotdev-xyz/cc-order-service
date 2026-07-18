package hidden

// This file is part of the grading suite for Problem 1. Its assertions are
// intentionally not summarized for the candidate — treat failures reported
// by `make test-hidden` as CI output only (expected vs. received), the same
// way an internal Amazon test pipeline would report a failing hidden case.

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"order-service/internal/domain"
	"order-service/internal/repository/postgres"
)

func newMockRepo(t *testing.T) (*postgres.OrderRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	repo := postgres.NewOrderRepository(db)
	return repo, mock, func() { db.Close() }
}

// Hidden Test 1: status/warehouseId filter values must never be concatenated
// into the SQL string. sqlmock is configured to only match queries that use
// placeholders; a query built via fmt.Sprintf with the raw value inlined
// will not match any expectation and this test will fail with "call to
// Query was not expected".
func TestHidden_RejectsSQLInjectionAttempt(t *testing.T) {
	repo, mock, closeFn := newMockRepo(t)
	defer closeFn()

	mock.MatchExpectationsInOrder(false)
	mock.ExpectQuery(`(?i)SELECT COUNT\(\*\)\s+FROM orders\s+WHERE`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`(?i)SELECT .*FROM orders\s+WHERE.*ORDER BY.*LIMIT.*OFFSET`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "customer_id", "warehouse_id", "seller_id", "status",
			"payment_status", "total_amount", "version", "created_at", "updated_at",
		}))

	malicious := domain.OrderStatus("PENDING'; DROP TABLE orders; --")
	_, _, err := repo.List(context.Background(), domain.OrderFilter{Status: &malicious}, domain.Pagination{Page: 1, Limit: 10})
	require.NoError(t, err, "query text did not use parameter placeholders for the status filter")

	require.NoError(t, mock.ExpectationsWereMet())
}

// Hidden Test 2: an unrecognized sortBy must fall back to created_at, and
// must never appear verbatim in the ORDER BY clause.
func TestHidden_SortByWhitelistFallback(t *testing.T) {
	repo, mock, closeFn := newMockRepo(t)
	defer closeFn()

	mock.MatchExpectationsInOrder(false)
	mock.ExpectQuery(`(?i)SELECT COUNT\(\*\)\s+FROM orders`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`(?i)ORDER BY created_at`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "customer_id", "warehouse_id", "seller_id", "status",
			"payment_status", "total_amount", "version", "created_at", "updated_at",
		}))

	_, _, err := repo.List(context.Background(), domain.OrderFilter{}, domain.Pagination{
		Page: 1, Limit: 10, SortBy: "status; DROP TABLE orders;--",
	})
	require.NoError(t, err, "unrecognized sortBy must fall back to created_at rather than reaching the query verbatim")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Hidden Test 3: pagination math — page 3, limit 10 must produce OFFSET 20.
func TestHidden_PaginationOffsetMath(t *testing.T) {
	repo, mock, closeFn := newMockRepo(t)
	defer closeFn()

	mock.MatchExpectationsInOrder(false)
	mock.ExpectQuery(`(?i)SELECT COUNT\(\*\)\s+FROM orders`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(35))
	mock.ExpectQuery(`(?i)LIMIT \$?\d+\s+OFFSET \$?\d+|OFFSET 20`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "customer_id", "warehouse_id", "seller_id", "status",
			"payment_status", "total_amount", "version", "created_at", "updated_at",
		}))

	_, total, err := repo.List(context.Background(), domain.OrderFilter{}, domain.Pagination{Page: 3, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 35, total, "total must reflect COUNT(*), unaffected by LIMIT/OFFSET")
}

// Hidden Test 4: with no filters set, no WHERE clause conditions should be
// present (COUNT should be a full-table count), verifying the "only add a
// condition for filters that are set" requirement.
func TestHidden_NoFiltersMeansNoWhereConditions(t *testing.T) {
	repo, mock, closeFn := newMockRepo(t)
	defer closeFn()

	mock.MatchExpectationsInOrder(false)
	mock.ExpectQuery(`(?i)^SELECT COUNT\(\*\) FROM orders\s*$`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`(?i)SELECT .*FROM orders\s+ORDER BY`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "customer_id", "warehouse_id", "seller_id", "status",
			"payment_status", "total_amount", "version", "created_at", "updated_at",
		}))

	_, _, err := repo.List(context.Background(), domain.OrderFilter{}, domain.Pagination{Page: 1, Limit: 10})
	require.NoError(t, err, "an empty filter must not add any WHERE conditions")
}

// Hidden Test 5 (QA-2026-0713): reported as "pagination skips/duplicates
// rows" — reproduces intermittently on datasets with many orders sharing
// the same createdAt (e.g. a bulk import). Not reliably reproducible with a
// single-row mock, so this test asserts the query shape that pagination
// correctness actually depends on, rather than the symptom.
func TestHidden_PaginationOrderingIsDeterministic(t *testing.T) {
	repo, mock, closeFn := newMockRepo(t)
	defer closeFn()

	mock.MatchExpectationsInOrder(false)
	mock.ExpectQuery(`(?i)SELECT COUNT\(\*\)\s+FROM orders`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`(?i)ORDER BY .+,\s*id\s+(ASC|DESC)`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "customer_id", "warehouse_id", "seller_id", "status",
			"payment_status", "total_amount", "version", "created_at", "updated_at",
		}))

	_, _, err := repo.List(context.Background(), domain.OrderFilter{}, domain.Pagination{Page: 1, Limit: 10})
	require.NoError(t, err, "ORDER BY needs a stable secondary key or ties on the primary sort column can shift rows between pages")
}

// Hidden Test 6 (TICKET-2026-0741): product wants GET /orders filterable by
// seller and payment status too. Same rules as every other filter here —
// parameterized, only added to the WHERE clause when set.
func TestHidden_SellerAndPaymentStatusFiltersAreParameterized(t *testing.T) {
	repo, mock, closeFn := newMockRepo(t)
	defer closeFn()

	mock.MatchExpectationsInOrder(false)
	mock.ExpectQuery(`(?i)SELECT COUNT\(\*\)\s+FROM orders\s+WHERE`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`(?is)SELECT .*FROM orders\s+WHERE.*(seller_id.*payment_status|payment_status.*seller_id).*ORDER BY`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "customer_id", "warehouse_id", "seller_id", "status",
			"payment_status", "total_amount", "version", "created_at", "updated_at",
		}))

	seller := "seller-42"
	paymentStatus := "PAID"
	_, _, err := repo.List(context.Background(), domain.OrderFilter{
		SellerID:      &seller,
		PaymentStatus: &paymentStatus,
	}, domain.Pagination{Page: 1, Limit: 10})
	require.NoError(t, err, "seller_id and payment_status filters must be added as parameterized WHERE conditions when set")
	require.NoError(t, mock.ExpectationsWereMet())
}
