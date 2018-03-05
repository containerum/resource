ALTER TABLE ingresses
  ADD COLUMN service_port INTEGER CHECK (service_port BETWEEN 1 AND 65535),
  ADD COLUMN path TEXT NOT NULL DEFAULT '/';

UPDATE ingresses i
SET i.service_port = (SELECT sp.port
                    FROM service_ports sp
                    WHERE (sp.service_id,sp.protocol) = (i.service_id, 'TCP')
                    UNION ALL
                    SELECT NULL
                    FETCH FIRST 1 ROW ONLY);

DELETE FROM ingresses WHERE service_port IS NULL;

ALTER TABLE ingresses
  ALTER COLUMN service_port SET NOT NULL;