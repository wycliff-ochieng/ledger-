-- +goose Up
SELECT 'up SQL query';

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. Enum Types
CREATE TYPE account_type AS ENUM ('ASSETS', 'LIABILITY', 'EQUITY', 'REVENUE', 'EXPENSE');
CREATE TYPE entry_direction AS ENUM ('DEBIT', 'CREDIT');
CREATE TYPE audit_status AS ENUM ('PENDING', 'BATCHED', 'FAILED');

-- 2. Ledgers
CREATE TABLE ledgers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Accounts
CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ledger_id UUID NOT NULL REFERENCES ledgers(id),
    name VARCHAR(255) NOT NULL,
    type account_type NOT NULL,
    normal_balance entry_direction NOT NULL,
    currency VARCHAR(3) NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_accounts_ledger ON accounts(ledger_id);

-- 4. Transactions (Append-Only)
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ledger_id UUID NOT NULL REFERENCES ledgers(id),
    idempotency_key VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    posted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tx_hash BYTEA NOT NULL,
    CONSTRAINT uq_transactions_idem UNIQUE (ledger_id, idempotency_key)
);
CREATE INDEX idx_transactions_ledger ON transactions(ledger_id, posted_at DESC);

-- 5. Ledger Entries (Append-Only)
CREATE TABLE ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE RESTRICT,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    amount BIGINT NOT NULL CHECK (amount > 0),
    direction entry_direction NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_entries_tx ON ledger_entries(transaction_id);
CREATE INDEX idx_entries_account ON ledger_entries(account_id);

-- 6. Merkle Epochs & Audit Outbox
CREATE TABLE merkle_epochs (
    epoch_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    epoch_number BIGSERIAL NOT NULL UNIQUE,
    root_hash BYTEA NOT NULL,
    previous_epoch BYTEA,
    transaction_count BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE audit_outbox (
    transaction_id UUID PRIMARY KEY REFERENCES transactions(id),
    leaf_hash BYTEA NOT NULL,
    epoch_id UUID REFERENCES merkle_epochs(epoch_id),
    status audit_status NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_audit_outbox_pending ON audit_outbox(created_at ASC) WHERE status = 'PENDING';

-- -----------------------------------------------------------------------------
-- IMMUTABILITY & DOUBLE-ENTRY TRIGGER GUARDS
-- -----------------------------------------------------------------------------

-- Guard 1: Hard ban on UPDATE or DELETE
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION forbid_mutation()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Cryptographic Ledger Rule: Records in table "%" are immutable and cannot be updated or deleted.', TG_TABLE_NAME;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER no_tamper_entries
BEFORE UPDATE OR DELETE ON ledger_entries
FOR EACH ROW EXECUTE FUNCTION forbid_mutation();

CREATE TRIGGER no_tamper_transactions
BEFORE UPDATE OR DELETE ON transactions
FOR EACH ROW EXECUTE FUNCTION forbid_mutation();

-- Guard 2: Deferrable Constraint Trigger enforcing Sum(Debits) == Sum(Credits)
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION enforce_zero_sum_balance()
RETURNS TRIGGER AS $$
DECLARE
    net_sum BIGINT;
    entry_count INT;
BEGIN
    SELECT 
        COUNT(*),
        COALESCE(SUM(CASE WHEN direction = 'DEBIT' THEN amount ELSE -amount END), 0)
    INTO entry_count, net_sum
    FROM ledger_entries
    WHERE transaction_id = NEW.transaction_id;

    IF entry_count < 2 THEN
        RAISE EXCEPTION 'Transaction % must have at least 2 entries', NEW.transaction_id;
    END IF;

    IF net_sum <> 0 THEN
        RAISE EXCEPTION 'Double-entry failure on Transaction %: net sum is % (must be 0)', NEW.transaction_id, net_sum;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE CONSTRAINT TRIGGER trg_enforce_zero_sum
AFTER INSERT ON ledger_entries
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION enforce_zero_sum_balance();


-- +goose Down
SELECT 'down SQL query';
