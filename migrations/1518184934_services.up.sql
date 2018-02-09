CREATE TYPE SERVICE_TYPE AS ENUM (
  'external',
  'internal'
);

CREATE TABLE IF NOT EXISTS services (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  deploy_id UUID NOT NULL REFERENCES deployments (id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  type SERVICE_TYPE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'UTC')
);

CREATE OR REPLACE FUNCTION check_unique_service_name() RETURNS TRIGGER AS $check_unique_service_name$
  DECLARE
    service_namespace UUID;
  BEGIN
    SELECT ns_id
    INTO service_namespace
    FROM deployments
    WHERE deploy_id = NEW.deploy_id;

    IF EXISTS(
      SELECT 1
      FROM deployments d
      JOIN services s ON d.id = s.deploy_id
      WHERE d.ns_id = service_namespace AND s.name = NEW.name
    ) THEN
      RAISE EXCEPTION 'service with name % already exists in namespace, can`t add', NEW.name;
    END IF;

    RETURN NEW;
  END;
$check_unique_service_name$ LANGUAGE plpgsql;

CREATE TRIGGER check_unique_service_name BEFORE INSERT OR UPDATE ON services
  FOR EACH ROW EXECUTE PROCEDURE check_unique_service_name();