CREATE TABLE IF NOT EXISTS env_vars (
  env_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  container_id UUID NOT NULL,
  name TEXT NOT NULL,
  value TEXT NOT NULL,

  FOREIGN KEY (container_id) REFERENCES containers (id) ON DELETE CASCADE,
  UNIQUE (container_id, name)
);