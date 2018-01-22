BEGIN TRANSACTION;
ALTER TABLE accesses
  ALTER COLUMN limited DROP NOT NULL;
ALTER TABLE accesses
  ADD CONSTRAINT accesses_check1 CHECK ( user_id <> owner_user_id AND limited IS NULL OR user_id = owner_user_id ) NOT VALID;
ALTER TABLE accesses
  ADD CONSTRAINT accesses_check CHECK ( user_id = owner_user_id AND limited IS NOT NULL OR user_id <> owner_user_id ) NOT VALID;
ALTER TABLE accesses
  DROP COLUMN IF EXISTS new_access_level;
COMMIT TRANSACTION;
