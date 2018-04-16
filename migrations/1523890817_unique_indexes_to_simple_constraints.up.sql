DROP INDEX IF EXISTS port_domain_index;
ALTER TABLE service_ports
  ADD CONSTRAINT unique_port_domain UNIQUE (port, protocol, domain_id) DEFERRABLE INITIALLY DEFERRED;

DROP INDEX IF EXISTS non_persistent_vols_index;
ALTER TABLE volumes
  ADD CONSTRAINT unique_ns_id UNIQUE (ns_id) DEFERRABLE INITIALLY DEFERRED;