DROP TRIGGER IF EXISTS check_unique_service_name ON services;

CREATE TRIGGER check_unique_service_name BEFORE INSERT ON services
  FOR EACH ROW EXECUTE PROCEDURE check_unique_service_name();