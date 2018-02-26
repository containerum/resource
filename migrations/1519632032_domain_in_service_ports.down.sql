DROP INDEX IF EXISTS port_domain_index;

ALTER TABLE service_ports
  DROP COLUMN IF EXISTS domain_id;
