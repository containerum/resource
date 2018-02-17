CREATE TABLE IF NOT EXISTS endpoints (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  namespace_id UUID NOT NULL REFERENCES namespaces (id) ON DELETE CASCADE,
  storage_id UUID NOT NULL REFERENCES storages (id) ON DELETE CASCADE,
  service_exists BOOLEAN NOT NULL,

  UNIQUE (namespace_id, storage_id)
);

-- remove permissions and endpoints if resource marked as deleted
CREATE OR REPLACE FUNCTION namespace_marked_deleted() RETURNS TRIGGER AS $namespace_marked_deleted$
BEGIN
  IF NEW.deleted = TRUE THEN
    DELETE FROM permissions WHERE resource_id = OLD.id;
    DELETE FROM endpoints WHERE namespace_id = OLD.id;
  END IF;
  RETURN NEW;
END;
$namespace_marked_deleted$ LANGUAGE plpgsql;