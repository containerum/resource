ALTER TABLE service_ports
  DROP CONSTRAINT IF EXISTS unique_port_domain;
CREATE UNIQUE INDEX port_domain_index ON service_ports (port, protocol, domain_id) WHERE domain_id IS NOT NULL;

ALTER TABLE service_ports
  DROP CONSTRAINT IF EXISTS unique_ns_id;
CREATE UNIQUE INDEX non_persistent_vols_index ON volumes (ns_id) WHERE (ns_id IS NOT NULL);

