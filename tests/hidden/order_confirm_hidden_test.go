package hidden

// Hidden tests for Problem 4 (Add Background Worker): OrderRepository.
// ConfirmOrder must be an idempotent, conditional PENDING -> CONFIRMED
// transition.

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestHidden_ConfirmOrder_OnlyTransitionsFromPending(t *testing.T) {
	repo, mock, closeFn := newMockRepo(t)
	defer closeFn()

	mock.ExpectExec(`(?i)UPDATE orders SET status = 'CONFIRMED'.*WHERE.*status = 'PENDING'`).
		WithArgs("order-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.ConfirmOrder(context.Background(), "order-1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHidden_ConfirmOrder_ZeroRowsIsNotAnError(t *testing.T) {
	repo, mock, closeFn := newMockRepo(t)
	defer closeFn()

	// Simulates calling ConfirmOrder a second time for an order that's
	// already CONFIRMED — the conditional UPDATE matches zero rows, and
	// that must NOT surface as an error.
	mock.ExpectExec(`(?i)UPDATE orders SET status = 'CONFIRMED'.*WHERE.*status = 'PENDING'`).
		WithArgs("order-1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.ConfirmOrder(context.Background(), "order-1")
	require.NoError(t, err, "zero rows affected means the order was already confirmed — that's success, not failure")
	require.NoError(t, mock.ExpectationsWereMet())
}
