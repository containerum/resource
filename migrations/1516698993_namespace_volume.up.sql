CREATE TABLE IF NOT EXISTS namespace_volume (
  ns_id UUID NOT NULL,
  vol_id UUID NOT NULL,

  FOREIGN KEY (ns_id) REFERENCES namespaces (id) ON DELETE CASCADE,
  FOREIGN KEY (vol_id) REFERENCES volumes (id) ON DELETE CASCADE
);