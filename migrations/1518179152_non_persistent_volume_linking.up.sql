ALTER TABLE volumes
  DROP COLUMN is_persistent,
  ADD COLUMN ns_id UUID DEFAULT NULL REFERENCES namespaces (id); -- if ns_id is not null, volume is non-persistent

CREATE UNIQUE INDEX non_persistent_vols_index ON volumes (ns_id) WHERE ns_id IS NOT NULL;