-- Add soft delete to wallets
ALTER TABLE wallets ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Drop the column-level unique constraint on user_id (it blocks creating a
-- new wallet after the old one is soft-deleted).
-- The constraint was created implicitly as "wallets_user_id_key".
ALTER TABLE wallets DROP CONSTRAINT IF EXISTS wallets_user_id_key;

-- Replace with a partial unique index: only one ACTIVE wallet per user.
CREATE UNIQUE INDEX IF NOT EXISTS uidx_wallets_user_id_active
    ON wallets (user_id)
    WHERE deleted_at IS NULL;

-- Index for quickly finding active wallet by user_id (covering query pattern).
CREATE INDEX IF NOT EXISTS idx_wallets_user_id_active
    ON wallets (user_id)
    WHERE deleted_at IS NULL;

-- Index for soft-deleted wallets (useful for audit / recovery queries).
CREATE INDEX IF NOT EXISTS idx_wallets_deleted_at
    ON wallets (deleted_at)
    WHERE deleted_at IS NOT NULL;
