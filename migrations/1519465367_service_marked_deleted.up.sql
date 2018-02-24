CREATE OR REPLACE FUNCTION service_marked_deleted() RETURNS TRIGGER AS $service_marked_deleted$
BEGIN
  IF NEW.deleted = TRUE THEN
    DELETE FROM permissions WHERE resource_id = OLD.id AND kind IN ('extservice', 'intservice');
  END IF;
  RETURN NEW;
END;
$service_marked_deleted$ LANGUAGE plpgsql;

CREATE TRIGGER service_marked_deleted BEFORE UPDATE ON services
  FOR EACH ROW EXECUTE PROCEDURE service_marked_deleted();