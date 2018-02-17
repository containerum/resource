DROP TABLE IF EXISTS endpoints;

-- remove permissions if resource marked as deleted
CREATE OR REPLACE FUNCTION namespace_marked_deleted() RETURNS TRIGGER AS $namespace_marked_deleted$
BEGIN
  IF NEW.deleted = TRUE THEN
    DELETE FROM permissions WHERE resource_id = OLD.id;
  END IF;
  RETURN NEW;
END;
$namespace_marked_deleted$ LANGUAGE plpgsql;