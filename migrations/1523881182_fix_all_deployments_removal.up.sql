CREATE OR REPLACE FUNCTION namespace_marked_deleted() RETURNS TRIGGER AS $namespace_marked_deleted$
BEGIN
  IF NEW.deleted = TRUE THEN
    DELETE FROM permissions WHERE resource_id = OLD.id;
    UPDATE deployments SET deleted = TRUE, delete_time = now() WHERE ns_id = OLD.id;
    DELETE FROM endpoints WHERE namespace_id = OLD.id;
  END IF;
  RETURN NEW;
END;
$namespace_marked_deleted$ LANGUAGE plpgsql;
