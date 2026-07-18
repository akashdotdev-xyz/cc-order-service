CREATE TABLE IF NOT EXISTS orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id     UUID NOT NULL,
    warehouse_id    UUID NOT NULL,
    seller_id       UUID NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    payment_status  VARCHAR(20) NOT NULL DEFAULT 'UNPAID',
    total_amount    BIGINT NOT NULL,
    version         INTEGER NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status);
CREATE INDEX IF NOT EXISTS idx_orders_warehouse_id ON orders (warehouse_id);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders (created_at);
