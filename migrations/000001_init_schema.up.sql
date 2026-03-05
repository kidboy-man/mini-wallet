-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username        VARCHAR(50) NOT NULL,
    hashed_password VARCHAR(255) NOT NULL,
    version         INTEGER NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- Case-insensitive unique index on username (active users only)
CREATE UNIQUE INDEX IF NOT EXISTS uidx_users_username
    ON users (LOWER(username))
    WHERE deleted_at IS NULL;

-- Wallets table (1-to-1 with users)
CREATE TABLE IF NOT EXISTS wallets (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE RESTRICT,
    balance       DECIMAL(20,4) NOT NULL DEFAULT 0.0000,
    locked_amount DECIMAL(20,4) NOT NULL DEFAULT 0.0000,
    version       INTEGER NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_wallets_balance_non_negative       CHECK (balance >= 0),
    CONSTRAINT chk_wallets_locked_non_negative        CHECK (locked_amount >= 0),
    CONSTRAINT chk_wallets_available_non_negative     CHECK (balance >= locked_amount)
);

-- Transactions table (immutable ledger)
CREATE TABLE IF NOT EXISTS transactions (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    from_id               UUID REFERENCES wallets(id) ON DELETE RESTRICT,
    to_id                 UUID REFERENCES wallets(id) ON DELETE RESTRICT,
    reference_id          VARCHAR(100),
    parent_transaction_id UUID REFERENCES transactions(id) ON DELETE RESTRICT,
    action                VARCHAR(20) NOT NULL,
    status                VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    amount                DECIMAL(20,4) NOT NULL,
    version               INTEGER NOT NULL DEFAULT 1,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_transactions_action         CHECK (action IN ('TOPUP','WITHDRAW','TRANSFER','REFUND')),
    CONSTRAINT chk_transactions_status         CHECK (status IN ('PENDING','SUCCESS','FAILED','REVERSED')),
    CONSTRAINT chk_transactions_amount_positive CHECK (amount > 0),
    CONSTRAINT chk_topup_no_sender             CHECK (action != 'TOPUP' OR from_id IS NULL),
    CONSTRAINT chk_withdraw_no_receiver        CHECK (action != 'WITHDRAW' OR to_id IS NULL)
);

-- Idempotency: unique (from_id, reference_id) for debits
CREATE UNIQUE INDEX IF NOT EXISTS uidx_transactions_reference
    ON transactions (from_id, reference_id)
    WHERE from_id IS NOT NULL AND reference_id IS NOT NULL;

-- Indexes for query patterns
CREATE INDEX IF NOT EXISTS idx_transactions_from_id
    ON transactions (from_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_transactions_to_id
    ON transactions (to_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_transactions_status_pending
    ON transactions (status)
    WHERE status = 'PENDING';

CREATE INDEX IF NOT EXISTS idx_transactions_parent_id
    ON transactions (parent_transaction_id)
    WHERE parent_transaction_id IS NOT NULL;
