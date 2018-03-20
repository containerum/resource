DROP TRIGGER IF EXISTS check_unique_service_name ON services;

CREATE TRIGGER check_unique_service_name BEFORE INSERT OR UPDATE ON services
  FOR EACH ROW EXECUTE PROCEDURE check_unique_service_name();
