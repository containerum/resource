CREATE TYPE PROTOCOL AS ENUM (
  'tcp',
  'udp'
);

CREATE TABLE IF NOT EXISTS service_ports (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  service_id UUID NOT NULL REFERENCES services (id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  port INTEGER NOT NULL CHECK (port BETWEEN 1 AND 65535),
  target_port INTEGER CHECK (target_port IS NULL OR target_port BETWEEN 1 AND 65535),
  protocol PROTOCOL NOT NULL DEFAULT 'tcp'::PROTOCOL,

  UNIQUE (service_id, name)
);
