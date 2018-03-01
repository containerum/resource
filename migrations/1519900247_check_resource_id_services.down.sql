CREATE OR REPLACE FUNCTION check_resource_id() RETURNS TRIGGER AS $check_resource_id$
BEGIN
  CASE NEW.kind
    WHEN 'namespace' THEN
    IF NOT EXISTS(SELECT 1 FROM namespaces WHERE id = NEW.resource_id) THEN
      RAISE EXCEPTION '% must be referenced to existing namespace id', NEW.resource_id;
    END IF;
    WHEN 'volume' THEN
    IF NOT EXISTS(SELECT 1 FROM volumes WHERE id = NEW.resource_id) THEN
      RAISE EXCEPTION '% must be referenced to existing volume id', NEW.resource_id;
    END IF;
  END CASE;
  RETURN NEW;
END;
$check_resource_id$ LANGUAGE plpgsql;