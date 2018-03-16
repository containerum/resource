CREATE OR REPLACE FUNCTION check_permissions() RETURNS TRIGGER AS $check_permissions$
DECLARE
  owner_access_level PERMISSION_STATUS;
BEGIN
  IF NEW.user_id != NEW.owner_user_id THEN
    IF NEW.access_level > owner_access_level OR NEW.new_access_level > owner_access_level THEN
      RAISE EXCEPTION 'permission must be less or equal to owner`s one';
    END IF;
  END IF;
  RETURN NEW;
END;
$check_permissions$ LANGUAGE plpgsql;