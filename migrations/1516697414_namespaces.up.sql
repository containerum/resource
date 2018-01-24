CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS namespaces (
  -- generic resource params
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  create_time TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'UTC'),
  deleted BOOLEAN NOT NULL DEFAULT FALSE,
  delete_time TIMESTAMPTZ DEFAULT NULL,
  tariff_id UUID NOT NULL,
  -- namespace-specific params
  ram INTEGER NOT NULL,
  cpu INTEGER NOT NULL,
  max_ext_services INTEGER NOT NULL,
  max_int_services INTEGER NOT NULL,
  max_traffic INTEGER NOT NULL,

  CHECK (ram > 0),
  CHECK (cpu > 0),
  CHECK (max_ext_services > 0),
  CHECK (max_int_services > 0),
  CHECK (max_traffic > 0)
);