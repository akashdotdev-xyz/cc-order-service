package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"order-service/internal/domain"
	"order-service/internal/service"
	"order-service/pkg/httputil"
)

type OrderHandler struct {
	svc *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

// ListOrders handles GET /orders.
//
// Supported query parameters:
//
//	status         - exact match, one of PENDING/CONFIRMED/SHIPPED/DELIVERED/CANCELLED
//	warehouseId    - exact match
//	createdAfter   - RFC3339 timestamp, inclusive lower bound
//	createdBefore  - RFC3339 timestamp, exclusive upper bound
//	page           - 1-indexed page number (default 1)
//	limit          - page size (default 20, max 100)
//	sortBy         - one of createdAt/totalAmount/status (default createdAt)
//	sortOrder      - asc|desc (default desc)
func (h *OrderHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	status, err := domain.ParseOrderStatus(q.Get("status"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid status")
		return
	}

	createdAfter, err := parseOptionalTime(q, "createdAfter")
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid createdAfter")
		return
	}

	createdBefore, err := parseOptionalTime(q, "createdBefore")
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid createdBefore")
		return
	}

	page, err := parseOptionalInt(q, "page")
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid page")
		return
	}

	limit, err := parseOptionalInt(q, "limit")
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid limit")
		return
	}

	filter := domain.OrderFilter{
		Status:        status,
		WarehouseID:   optionalString(q, "warehouseId"),
		SellerID:      optionalString(q, "sellerId"),
		PaymentStatus: optionalString(q, "paymentStatus"),
		CreatedAfter:  createdAfter,
		CreatedBefore: createdBefore,
	}

	// page/limit stay 0 (unset) when absent from the query string — the
	// service layer owns defaulting, so we don't duplicate that here.
	pagination := domain.Pagination{
		Page:      page,
		Limit:     limit,
		SortBy:    q.Get("sortBy"),
		SortOrder: q.Get("sortOrder"),
	}

	result, err := h.svc.ListOrders(r.Context(), filter, pagination)
	if err != nil {
		if isValidationErr(err) {
			httputil.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, result)
}

// CreateOrder handles POST /orders.
//
// TODO(candidate): implement this handler.
//
// Requirements:
//   - Decode the JSON request body into a domain.CreateOrderRequest. On a
//     malformed body, respond 400.
//   - Call h.svc.CreateOrder.
//   - On a validation error (isValidationErr), respond 400 with the error
//     message.
//   - On errors.Is(err, domain.ErrInsufficientInventory), respond 409.
//   - On any other error, respond 500.
//   - On success, respond 201 Created with the created *domain.Order as
//     JSON.
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {

	var req domain.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	order, err := h.svc.CreateOrder(r.Context(), req)
	if err != nil {
		if isValidationErr(err) {
			httputil.WriteError(w, http.StatusBadRequest, err.Error())
			return
		} else if errors.Is(err, domain.ErrInsufficientInventory) {
			httputil.WriteError(w, http.StatusConflict, err.Error())
			return
		}

		httputil.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, order)
}

// optionalString returns nil when the query param is absent, otherwise a
// pointer to its raw value.
func optionalString(q url.Values, key string) *string {
	v := q.Get(key)
	if v == "" {
		return nil
	}
	return &v
}

// parseOptionalTime returns (nil, nil) when the query param is absent,
// otherwise the parsed RFC3339 timestamp.
func parseOptionalTime(q url.Values, key string) (*time.Time, error) {
	v := q.Get(key)
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// parseOptionalInt returns (0, nil) when the query param is absent,
// otherwise the parsed integer. 0 doubles as the "unset" sentinel the
// service layer defaults on.
func parseOptionalInt(q url.Values, key string) (int, error) {
	v := q.Get(key)
	if v == "" {
		return 0, nil
	}
	return strconv.Atoi(v)
}

func isValidationErr(err error) bool {
	return errors.Is(err, service.ErrInvalidPage) ||
		errors.Is(err, service.ErrInvalidLimit) ||
		errors.Is(err, service.ErrInvalidDateRange) ||
		errors.Is(err, service.ErrInvalidOrderStatus) ||
		errors.Is(err, service.ErrInvalidSortOrder) ||
		errors.Is(err, service.ErrMissingCustomerID) ||
		errors.Is(err, service.ErrMissingWarehouseID) ||
		errors.Is(err, service.ErrMissingSellerID) ||
		errors.Is(err, service.ErrEmptyItems) ||
		errors.Is(err, service.ErrInvalidItemQuantity)
}
