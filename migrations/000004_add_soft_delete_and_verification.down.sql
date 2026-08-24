DROP INDEX IF EXISTS idx_users_is_verified;
DROP INDEX IF EXISTS idx_users_deleted_at;

ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE users DROP COLUMN IF EXISTS verified_at;
ALTER TABLE users DROP COLUMN IF EXISTS is_verified;
