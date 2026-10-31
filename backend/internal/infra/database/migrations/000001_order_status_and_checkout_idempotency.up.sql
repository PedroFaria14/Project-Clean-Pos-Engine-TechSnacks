-- Compatible with existing GORM table names and a fresh database.
CREATE TABLE IF NOT EXISTS order_models (
    id text PRIMARY KEY,
    total_cents bigint NOT NULL
);
CREATE TABLE IF NOT EXISTS order_item_models (
    id bigserial PRIMARY KEY,
    order_id text REFERENCES order_models(id),
    product_id text,
    product_name text,
    quantity bigint,
    price_cents bigint
);

ALTER TABLE order_models ADD COLUMN IF NOT EXISTS status text;
-- Approved legacy rule: persisted checkouts were awaiting payment, never assume paid.
UPDATE order_models SET status = 'AWAITING_PAYMENT' WHERE status IS NULL OR status = '';
ALTER TABLE order_models ALTER COLUMN status SET DEFAULT 'AWAITING_PAYMENT';
ALTER TABLE order_models ALTER COLUMN status SET NOT NULL;
ALTER TABLE order_models ADD CONSTRAINT order_models_status_check
    CHECK (status IN ('OPEN', 'AWAITING_PAYMENT', 'COMPLETED'));

-- This table scopes keys to POST /checkout. Only committed results are recorded.
CREATE TABLE checkout_idempotency_records (
    key varchar(128) COLLATE "C" PRIMARY KEY,
    request_hash char(64) NOT NULL,
    order_id text NOT NULL REFERENCES order_models(id),
    total_cents bigint NOT NULL CHECK (total_cents >= 0),
    status text NOT NULL CHECK (status = 'AWAITING_PAYMENT'),
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);
