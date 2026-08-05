-- Initial schema for payments service

CREATE TABLE IF NOT EXISTS payments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key TEXT NOT NULL UNIQUE,
    amount          BIGINT NOT NULL CHECK (amount > 0),
    currency        TEXT NOT NULL DEFAULT 'NGN',
    status          TEXT NOT NULL DEFAULT 'pending',
    failure_reason  TEXT,
    customer_id     TEXT NOT NULL,
    merchant_id     TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_payments_customer_id ON payments (customer_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payments_merchant_id ON payments (merchant_id, created_at DESC);
