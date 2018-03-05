DROP TRIGGER IF EXISTS check_service_port ON ingresses;

DROP FUNCTION IF EXISTS check_service_port();

ALTER TABLE ingresses
  DROP COLUMN IF EXISTS service_port,
  DROP COLUMN IF EXISTS "path";
