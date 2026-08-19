-- accounts table

CREATE TABLE accounts(
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL, -- 'asset', 'liability', 'equity', 'revenue', 'expense'
    currency TEXT NOT NULL DEFAULT 'NGN',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


-- ledger_entries table
CREATE TABLE ledger_entries (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_id TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


CREATE TABLE ledger_lines (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id TEXT NOT NULL REFERENCES ledger_entries(id),
    account_id TEXT NOT NULL REFERENCES accounts(id),
    amount BIGINT NOT NULL CHECK (amount > 0),
    direction TEXT NOT NULL CHECK (direction IN ('debit', 'credit')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);



