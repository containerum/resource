CREATE TABLE IF NOT EXISTS volumes (
  -- generic resource params
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  create_time TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'UTC'),
  deleted BOOLEAN NOT NULL DEFAULT FALSE,
  delete_time TIMESTAMPTZ DEFAULT NULL,
  tariff_id UUID NOT NULL,
  -- volume-specific params
  active BOOLEAN NOT NULL DEFAULT FALSE,
  capacity INTEGER NOT NULL,
  replicas INTEGER NOT NULL,
  is_persistent BOOLEAN NOT NULL,

  CHECK (capacity > 0),
  CHECK (replicas >= 2)
);