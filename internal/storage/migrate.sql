-- Enum for transaction status
CREATE TYPE tx_status AS ENUM ('PENDING', 'SUCCESSFUL', 'FAILED');

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

-- Indexes we'll actually use
CREATE INDEX idx_transactions_external_id   ON transactions(external_id);
CREATE INDEX idx_transactions_status        ON transactions(status);
CREATE INDEX idx_transactions_received_at   ON transactions(received_at DESC);