CREATE OR REPLACE FUNCTION deployment_marked_deleted() RETURNS TRIGGER AS $deployment_marked_deleted$
  BEGIN
    IF NEW.deleted = TRUE THEN
      DELETE FROM containers WHERE depl_id = NEW.id;
    END IF;
    RETURN NEW;
  END;
$deployment_marked_deleted$ LANGUAGE plpgsql;

CREATE TRIGGER deployment_marked_deleted BEFORE UPDATE ON deployments
  FOR EACH ROW EXECUTE PROCEDURE deployment_marked_deleted();