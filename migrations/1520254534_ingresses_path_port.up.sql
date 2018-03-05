ALTER TABLE ingresses
  ADD COLUMN service_port INTEGER CHECK (service_port BETWEEN 1 AND 65535),
  ADD COLUMN path TEXT NOT NULL DEFAULT '/';

UPDATE ingresses i
SET i.service_port = (SELECT sp.port
                    FROM service_ports sp
                    WHERE (sp.service_id,sp.protocol) = (i.service_id, 'TCP'::PROTOCOL)
                    UNION ALL
                    SELECT NULL
                    FETCH FIRST 1 ROW ONLY);

DELETE FROM ingresses WHERE service_port IS NULL;

ALTER TABLE ingresses
  ALTER COLUMN service_port SET NOT NULL;

CREATE OR REPLACE FUNCTION check_service_port() RETURNS TRIGGER AS $check_service_port$
DECLARE
  s_type SERVICE_TYPE;
BEGIN
  SELECT s.type INTO s_type FROM services s WHERE id = NEW.service_id;
  IF s_type != 'external'::SERVICE_TYPE THEN
    RAISE EXCEPTION 'service % is not external', NEW.service_id;
  END IF;

  IF NOT EXISTS(
    SELECT 1 FROM service_ports sp WHERE (sp.service_id, sp.protocol) = (NEW.service_id, 'TCP'::PROTOCOL)
  ) THEN
    RAISE EXCEPTION 'TCP port % not found in service', NEW.service_port;
  END IF;

  RETURN NEW;
END;
$check_service_port$ LANGUAGE plpgsql;

CREATE TRIGGER check_service_port BEFORE INSERT OR UPDATE ON ingresses
  FOR EACH ROW EXECUTE PROCEDURE check_service_port();