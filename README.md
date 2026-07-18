# order-service

Order management backend service.

## Stack

Go, PostgreSQL, REST.

## Running locally

```
docker-compose up --build
```

## Current Task: Problem 1 — Implement `GET /orders`

Implement listing orders with filters, pagination, and sorting.

```
GET /orders
```

Query parameters:

| param          | type   | notes                                             |
|----------------|--------|----------------------------------------------------|
| status         | string | one of PENDING/CONFIRMED/SHIPPED/DELIVERED/CANCELLED |
| warehouseId    | string | exact match                                       |
| createdAfter   | string | RFC3339 timestamp, inclusive lower bound           |
| createdBefore  | string | RFC3339 timestamp, exclusive upper bound           |
| page           | int    | 1-indexed, default 1                              |
| limit          | int    | default 20, max 100                               |
| sortBy         | string | createdAt \| totalAmount \| status, default createdAt |
| sortOrder      | string | asc \| desc, default desc                          |

### Files to modify

- `internal/api/handlers/order_handler.go`
- `internal/service/order_service.go`
- `internal/repository/postgres/order_repository.go`

Do not change `internal/repository/order_repository.go` (the interface),
`internal/domain/order.go`, or anything under `tests/`.

### Running tests

```
make test-unit
make test-integration
make test-hidden
```
