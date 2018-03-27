ALTER TABLE service_ports
  DROP CONSTRAINT IF EXISTS service_ports_target_port_check,
  DROP CONSTRAINT IF EXISTS service_ports_port_check,
  ADD CHECK (target_port BETWEEN 1 AND 65535),
  ADD CHECK (port IS NULL OR target_port BETWEEN 1 AND 65535);