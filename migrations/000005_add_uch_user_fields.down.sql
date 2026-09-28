DROP INDEX IF EXISTS idx_users_id_number;
ALTER TABLE users DROP COLUMN IF EXISTS affiliation;
ALTER TABLE users DROP COLUMN IF EXISTS id_number;
