DROP INDEX IF EXISTS idx_wallets_deleted_at;
DROP INDEX IF EXISTS idx_wallets_user_id_active;
DROP INDEX IF EXISTS uidx_wallets_user_id_active;

-- Restore the original column-level unique constraint.
ALTER TABLE wallets ADD CONSTRAINT wallets_user_id_key UNIQUE (user_id);

ALTER TABLE wallets DROP COLUMN IF EXISTS deleted_at;
