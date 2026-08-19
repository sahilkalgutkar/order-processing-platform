CREATE TABLE IF NOT EXISTS orders (
    id          UUID PRIMARY KEY,
    customer_id TEXT NOT NULL,
    item_sku    TEXT NOT NULL,
    quantity    INTEGER NOT NULL CHECK (quantity > 0),
    status      TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders (customer_id);
