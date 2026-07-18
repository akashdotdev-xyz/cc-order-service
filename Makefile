.PHONY: run build test test-unit test-integration test-hidden migrate-up migrate-down

run:
	go run ./cmd

build:
	go build -o bin/order-service ./cmd

test:
	go test ./... -v

test-unit:
	go test ./tests/unit/... -v

test-integration:
	go test ./tests/integration/... -v

test-hidden:
	go test ./tests/hidden/... -v

migrate-up:
	psql "$$DATABASE_URL" -f db/migrations/0001_create_orders_table.up.sql

migrate-down:
	psql "$$DATABASE_URL" -f db/migrations/0001_create_orders_table.down.sql
