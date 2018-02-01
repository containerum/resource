CREATE OR REPLACE FUNCTION set_limited() RETURNS TRIGGER AS $set_limited$
BEGIN
  IF OLD.new_access_level != NEW.new_access_level THEN
    NEW.limited := TRUE;
    NEW.access_level_change_time := (now() AT TIME ZONE 'UTC');
  END IF;
  RETURN NEW;
END;
$set_limited$ LANGUAGE plpgsql;

CREATE TRIGGER set_limited BEFORE UPDATE ON permissions
  FOR EACH ROW EXECUTE PROCEDURE set_limited();
