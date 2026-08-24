-- Transfers service schema
-- Each transfer tracks a money movement request between two parties,
-- coordinated across services using the Saga pattern.

CREATE TABLE IF NOT EXISTS transfers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key TEXT NOT NULL UNIQUE,

    -- Who is sending and who is receiving
    sender_id       TEXT NOT NULL,
    receiver_id     TEXT NOT NULL,

    -- Amount in kobo (BIGINT, smallest currency unit)
    amount          BIGINT NOT NULL CHECK (amount > 0),
    currency        TEXT NOT NULL DEFAULT 'NGN',

    -- Transfer lifecycle state (managed by Go code, not DB constraint)
    -- pending -> processing -> completed / failed / reversed
    status          TEXT NOT NULL DEFAULT 'pending',
    failure_reason  TEXT,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes for querying transfers by participant
CREATE INDEX IF NOT EXISTS idx_transfers_sender_id ON transfers (sender_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transfers_receiver_id ON transfers (receiver_id, created_at DESC);
