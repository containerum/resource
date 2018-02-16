DROP TRIGGER IF EXISTS check_permissions ON permissions;
DROP FUNCTION IF EXISTS check_permissions();

ALTER TABLE permissions
  DROP CONSTRAINT IF EXISTS check_perm;

-- check if we updating resource owner`s permissions
CREATE OR REPLACE FUNCTION check_update_permissions() RETURNS TRIGGER AS $check_update_permissions$
DECLARE
  owner_new_access_level PERMISSION_STATUS;
BEGIN
  IF NEW.owner_user_id != NEW.user_id THEN
    SELECT new_access_level
    INTO owner_new_access_level
    FROM permissions
    WHERE owner_user_id = user_id AND NEW.owner_user_id = user_id;
    -- check if permission lower or equal to owner`s
    IF NEW.new_access_level > owner_new_access_level THEN
      RAISE EXCEPTION 'new access level for non owner must be lower or equal to owner`s one';
    END IF;
  END IF;
  RETURN NEW;
END;
$check_update_permissions$ LANGUAGE plpgsql;

CREATE TRIGGER check_update_permissions BEFORE UPDATE ON permissions
  FOR EACH ROW EXECUTE PROCEDURE check_update_permissions();

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

-- insert permissions as owner`s one
CREATE OR REPLACE FUNCTION insert_owner_permissions() RETURNS TRIGGER AS $insert_owner_permissions$
BEGIN
  IF NEW.user_id != NEW.owner_user_id THEN -- rewrite permissions with owner`s one
    SELECT limited, access_level, new_access_level, access_level_change_time
    INTO NEW.limited, NEW.access_level, NEW.new_access_level, NEW.access_level_change_time
    FROM permissions
    WHERE owner_user_id = NEW.owner_user_id AND owner_user_id = user_id;
  END IF;
  RETURN NEW;
END;
$insert_owner_permissions$ LANGUAGE plpgsql;

CREATE TRIGGER insert_owner_permissions BEFORE INSERT ON permissions
  FOR EACH ROW EXECUTE PROCEDURE insert_owner_permissions();
