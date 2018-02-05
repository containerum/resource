CREATE TABLE IF NOT EXISTS volume_mounts (
  mount_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  container_id UUID NOT NULL,
  volume_id UUID NOT NULL,
  mount_path TEXT NOT NULL,
  sub_path TEXT,

  FOREIGN KEY (container_id) REFERENCES containters (id) ON DELETE CASCADE,
  FOREIGN KEY (volume_id) REFERENCES volumes (id) ON DELETE CASCADE,
  UNIQUE (container_id, mount_path)
);