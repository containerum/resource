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
