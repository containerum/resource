CREATE OR REPLACE FUNCTION remove_volume_mounts() RETURNS TRIGGER AS $remove_volume_mounts$
  BEGIN
    IF NEW.deleted = TRUE THEN
      DELETE FROM volume_mounts WHERE volume_id = OLD.id;
    END IF;
    RETURN NEW;
  END;
$remove_volume_mounts$ LANGUAGE plpgsql;

CREATE TRIGGER remove_volume_mounts BEFORE UPDATE ON volumes
  FOR EACH ROW EXECUTE PROCEDURE remove_volume_mounts();