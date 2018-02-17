DROP TRIGGER IF EXISTS insert_owner_permissions ON permissions;
DROP FUNCTION IF EXISTS insert_owner_permissions();

ALTER TYPE PERMISSION_STATUS RENAME TO PERMISSION_STATUS_OLD;
CREATE TYPE PERMISSION_STATUS AS ENUM (
  'none',
  'read',
  'readdelete',
  'write',
  'owner'
);
ALTER TABLE permissions
  ALTER COLUMN access_level DROP DEFAULT,
  ALTER COLUMN new_access_level DROP DEFAULT,
  ALTER COLUMN access_level SET DATA TYPE PERMISSION_STATUS USING access_level::TEXT::PERMISSION_STATUS,
  ALTER COLUMN new_access_level SET DATA TYPE PERMISSION_STATUS USING new_access_level::TEXT::PERMISSION_STATUS,
  ALTER COLUMN access_level SET DEFAULT 'owner',
  ALTER COLUMN new_access_level SET DEFAULT 'owner';
DROP TYPE PERMISSION_STATUS_OLD;

DROP TRIGGER IF EXISTS check_update_permissions ON permissions;
DROP FUNCTION IF EXISTS check_update_permissions();

UPDATE permissions SET access_level = 'write' WHERE owner_user_id != user_id AND access_level = 'owner';
UPDATE permissions SET new_access_level = 'write' WHERE owner_user_id != user_id AND new_access_level = 'owner';
ALTER TABLE permissions
  ADD CONSTRAINT check_perm CHECK (
    (owner_user_id != user_id AND access_level < 'owner' AND new_access_level < 'owner') OR
    (owner_user_id = user_id));

CREATE OR REPLACE FUNCTION check_permissions() RETURNS TRIGGER AS $check_permissions$
DECLARE
  owner_access_level PERMISSION_STATUS;
  owner_limited BOOLEAN;
BEGIN
  IF NEW.user_id != NEW.owner_user_id THEN
    SELECT access_level, limited
    INTO owner_access_level, owner_limited
    FROM permissions
    WHERE owner_user_id = NEW.owner_user_id AND owner_user_id = user_id;
    IF owner_limited THEN
      RAISE EXCEPTION 'limited owner can`t assign permissions';
    END IF;
    IF NEW.access_level > owner_access_level OR NEW.new_access_level > owner_access_level THEN
      RAISE EXCEPTION 'permission must be less or equal to owner`s one';
    END IF;
  END IF;
  RETURN NEW;
END;
$check_permissions$ LANGUAGE plpgsql;

CREATE TRIGGER check_permissions BEFORE INSERT OR UPDATE ON permissions
  FOR EACH ROW EXECUTE PROCEDURE check_permissions();
