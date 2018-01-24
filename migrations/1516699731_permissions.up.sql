CREATE TYPE RESOURCE_KIND AS ENUM (
  'namespace',
  'volume',
  'extservice',
  'intservice',
  'domain'
);

CREATE TYPE PERMISSION_STATUS AS ENUM (
  'owner',
  'read',
  'write',
  'readdelete',
  'none'
);

CREATE TABLE IF NOT EXISTS permissions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  kind RESOURCE_KIND NOT NULL,
  resource_id UUID,
  resource_label TEXT NOT NULL,
  owner_user_id UUID NOT NULL,
  create_time TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'UTC'),
  user_id UUID NOT NULL,
  access_level PERMISSION_STATUS NOT NULL,
  limited BOOLEAN NOT NULL DEFAULT FALSE,
  access_level_change_time TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'UTC'),
  new_access_level UUID NOT NULL,

  UNIQUE (kind, resource_id, resource_label, owner_user_id)
);

-- check if newly inserted permission for namespace or volume is referenced to existing namespace or volume
CREATE OR REPLACE FUNCTION check_resource_id() RETURNS TRIGGER AS $check_resource_id$
  BEGIN
    CASE NEW.kind
      WHEN 'namespace' THEN
      IF NOT EXISTS(SELECT 1 FROM namespaces WHERE id = NEW.resource_id) THEN
         RAISE EXCEPTION '% must be referenced to existing namespace id', NEW.resource_id;
      END IF;
      WHEN 'volume' THEN
      IF NOT EXISTS(SELECT 1 FROM volumes WHERE id = NEW.resource_id) THEN
        RAISE EXCEPTION '% must be referenced to existing volume id', NEW.resource_id;
      END IF;
    END CASE;
  END;
$check_resource_id$ LANGUAGE plpgsql;

CREATE TRIGGER check_resource_id BEFORE INSERT OR UPDATE ON permissions
  FOR EACH ROW EXECUTE PROCEDURE check_resource_id();

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
  END;
$check_update_permissions$ LANGUAGE plpgsql;

CREATE TRIGGER check_update_permissions BEFORE UPDATE ON permissions
  FOR EACH ROW EXECUTE PROCEDURE check_update_permissions();

CREATE OR REPLACE FUNCTION set_limited() RETURNS TRIGGER AS $set_limited$
  BEGIN
    IF OLD.new_access_level != NEW.new_access_level THEN
      NEW.limited := TRUE;
      NEW.access_level_change_time := (now() AT TIME ZONE 'UTC');
    END IF;
  END;
$set_limited$ LANGUAGE plpgsql;

CREATE TRIGGER set_limited AFTER UPDATE ON permissions
  FOR EACH ROW EXECUTE PROCEDURE set_limited();

-- insert permissions as owner`s one
CREATE OR REPLACE FUNCTION insert_owner_permissions() RETURNS TRIGGER AS $insert_owner_permissions$
  BEGIN
    IF NEW.user_id != NEW.owner_user_id THEN -- rewrite permissions with owner`s one
      SELECT limited, access_level, new_access_level, access_level_change_time
      INTO NEW.limited, NEW.access_level, NEW.new_access_level, NEW.access_level_change_time
      FROM permissions
      WHERE owner_user_id = NEW.owner_user_id AND owner_user_id = user_id;
    END IF;
  END;
$insert_owner_permissions$ LANGUAGE plpgsql;

CREATE TRIGGER insert_owner_permissions BEFORE INSERT ON permissions
  FOR EACH ROW EXECUTE PROCEDURE insert_owner_permissions();

-- remove all non owners access records if we remove owner`s access record
CREATE OR REPLACE FUNCTION remove_users_on_remove_owner() RETURNS TRIGGER AS $remove_users_on_remove_owner$
  BEGIN
    IF OLD.user_id = OLD.owner_user_id THEN
      DELETE FROM permissions WHERE user_id = OLD.owner_user_id;
    END IF;
  END;
$remove_users_on_remove_owner$ LANGUAGE plpgsql;

CREATE TRIGGER remove_users_on_remove_owner AFTER DELETE ON permissions
  FOR EACH ROW EXECUTE PROCEDURE remove_users_on_remove_owner();

-- simulate cascade removal
CREATE OR REPLACE FUNCTION remove_namespace_perms() RETURNS TRIGGER AS $remove_namespace_perms$
  BEGIN
    DELETE FROM permissions WHERE resource_id = OLD.id;
  END;
$remove_namespace_perms$ LANGUAGE plpgsql;

CREATE TRIGGER remove_namespace_perms BEFORE DELETE ON namespaces
  FOR EACH ROW EXECUTE PROCEDURE remove_namespace_perms();

CREATE OR REPLACE FUNCTION remove_volume_perms() RETURNS TRIGGER AS $remove_volume_perms$
  BEGIN
    DELETE FROM permissions WHERE resource_id = OLD.id;
  END;
$remove_volume_perms$ LANGUAGE plpgsql;

CREATE TRIGGER remove_volume_perms BEFORE DELETE ON volumes
  FOR EACH ROW EXECUTE PROCEDURE remove_volume_perms();

-- remove permissions if resource marked as deleted
CREATE OR REPLACE FUNCTION namespace_marked_deleted() RETURNS TRIGGER AS $namespace_marked_deleted$
  BEGIN
    IF NEW.deleted = TRUE THEN
      DELETE FROM permissions WHERE resource_id = OLD.id;
    END IF;
  END;
$namespace_marked_deleted$ LANGUAGE plpgsql;

CREATE TRIGGER namespace_marked_deleted BEFORE UPDATE ON namespaces
  FOR EACH ROW EXECUTE PROCEDURE namespace_marked_deleted();

CREATE OR REPLACE FUNCTION volume_marked_deleted() RETURNS TRIGGER AS $volume_marked_deleted$
BEGIN
  IF NEW.deleted = TRUE THEN
    DELETE FROM permissions WHERE resource_id = OLD.id;
  END IF;
END;
$volume_marked_deleted$ LANGUAGE plpgsql;

CREATE TRIGGER volume_marked_deleted BEFORE UPDATE ON volumes
  FOR EACH ROW EXECUTE PROCEDURE volume_marked_deleted()