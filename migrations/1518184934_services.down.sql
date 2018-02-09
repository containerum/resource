DROP TRIGGER IF EXISTS check_unique_service_name ON services;

DROP FUNCTION IF EXISTS check_unique_service_name();

DROP TABLE IF EXISTS services;

DROP TYPE IF EXISTS SERVICE_TYPE;