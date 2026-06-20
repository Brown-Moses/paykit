-- Enum for transaction status (idempotent creation)
DO $$ 
BEGIN 
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'tx_status') THEN
        CREATE TYPE tx_status AS ENUM ('PENDING', 'SUCCESSFUL', 'FAILED');
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS transactions (
    id              BIGSERIAL PRIMARY KEY,
    provider_tx_id  TEXT        NOT NULL UNIQUE,  -- idempotency key
    external_id     TEXT        NOT NULL,          -- maps to your order_id
    amount          NUMERIC(20, 2) NOT NULL,
    currency        VARCHAR(3)  NOT NULL DEFAULT 'RWF',
    status          tx_status   NOT NULL DEFAULT 'PENDING',
    payer_msisdn    TEXT,                          -- SHA-256 hashed
    raw_payload     JSONB,                         -- full webhook body, for auditing
    received_at     TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Merchants
CREATE TABLE IF NOT EXISTS merchants (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT        NOT NULL,
    api_key     TEXT        NOT NULL UNIQUE,
    webhook_url TEXT,
    active      BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add merchant_id to transactions (idempotent creation)
ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS merchant_id BIGINT REFERENCES merchants(id);

-- Indexes we'll actually use (idempotent creation)
CREATE INDEX IF NOT EXISTS idx_merchants_api_key ON merchants(api_key);
CREATE INDEX IF NOT EXISTS idx_transactions_merchant_id ON transactions(merchant_id);
CREATE INDEX IF NOT EXISTS idx_transactions_external_id ON transactions(external_id);
CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions(status);
CREATE INDEX IF NOT EXISTS idx_transactions_received_at ON transactions(received_at DESC);

-- Enum for delivery status (idempotent creation)
DO $$ 
BEGIN 
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'delivery_status') THEN
        CREATE TYPE delivery_status AS ENUM ('SUCCESS', 'FAILED', 'RETRYING');
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS delivery_logs (
    id              BIGSERIAL PRIMARY KEY,
    transaction_id  BIGINT      NOT NULL REFERENCES transactions(id),
    merchant_id     BIGINT      NOT NULL REFERENCES merchants(id),
    webhook_url     TEXT        NOT NULL,
    attempt         INT         NOT NULL DEFAULT 1,
    status          delivery_status NOT NULL,
    response_code   INT,                        -- HTTP status merchant returned
    error_message   TEXT,                       -- network error if any
    delivered_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_delivery_logs_transaction_id ON delivery_logs(transaction_id);
CREATE INDEX IF NOT EXISTS idx_delivery_logs_merchant_id ON delivery_logs(merchant_id);
