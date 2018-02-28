CREATE OR REPLACE FUNCTION check_unique_service_name() RETURNS TRIGGER AS $check_unique_service_name$
DECLARE
  service_namespace UUID;
BEGIN
  SELECT ns_id
  INTO service_namespace
  FROM deployments
  WHERE id = NEW.deploy_id;

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