ALTER TABLE service_ports
  ADD COLUMN domain_id UUID REFERENCES domains (id) ON DELETE CASCADE;

CREATE UNIQUE INDEX port_domain_index
ON service_ports (port, protocol, domain_id)
WHERE domain_id IS NOT NULL;
