CREATE TABLE IF NOT EXISTS containers (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  depl_id UUID NOT NULL,
  name TEXT NOT NULL,
  image TEXT NOT NULL,
  ram INTEGER NOT NULL,
  cpu INTEGER NOT NULL,

  FOREIGN KEY (depl_id) REFERENCES deployments (id) ON DELETE CASCADE,
  CHECK (ram > 0),
  CHECK (cpu > 0),
  CHECK (name != ''),
  CHECK (image != ''),
  UNIQUE (depl_id, name)
);