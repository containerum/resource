CREATE OR REPLACE FUNCTION check_services_quota() RETURNS TRIGGER AS $check_services_quota$
DECLARE
  max_external INTEGER;
  max_internal INTEGER;
  current_external INTEGER;
  current_internal INTEGER;
  service_ns_id UUID;
BEGIN
  SELECT ns.max_ext_services, ns.max_int_services, ns.id
  INTO max_external, max_internal, service_ns_id
  FROM deployments d
    JOIN namespaces ns ON d.ns_id = ns.id
  WHERE d.id = NEW.deploy_id;

  SELECT
    count(*) FILTER (WHERE s.type = 'external'),
    count(*) FILTER (WHERE s.type = 'internal')
  INTO current_external, current_internal
  FROM services s
    JOIN deployments d ON s.deploy_id = d.id
    JOIN namespaces ns ON d.ns_id = ns.id
  WHERE ns.id = service_ns_id;

  IF current_external >= max_external AND NEW.type = 'external' THEN
    RAISE EXCEPTION 'can`t add external service, quota exceeded';
  END IF;
  IF current_internal >= max_internal AND NEW.type = 'internal' THEN
    RAISE EXCEPTION 'can`t add internal service, quota exceeded';
  END IF;

  RETURN NEW;
END;
$check_services_quota$ LANGUAGE plpgsql;
