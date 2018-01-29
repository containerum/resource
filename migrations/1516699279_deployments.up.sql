CREATE TABLE IF NOT EXISTS deployments (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  ns_id UUID NOT NULL,
  create_time TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'UTC'),
  deleted BOOLEAN NOT NULL DEFAULT FALSE,
  delete_time TIMESTAMPTZ DEFAULT NULL,
  name TEXT NOT NULL,
  ram INTEGER NOT NULL,
  cpu INTEGER NOT NULL,
  image TEXT NOT NULL,

  FOREIGN KEY (ns_id) REFERENCES namespaces (id) ON DELETE CASCADE,
  CHECK (ram > 0),
  CHECK (cpu > 0),
  CHECK (name != ''),
  CHECK (image != '')
);