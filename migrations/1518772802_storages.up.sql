CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS storages (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name TEXT NOT NULL UNIQUE,
  used INTEGER NOT NULL DEFAULT 0 CHECK (used >= 0),
  size INTEGER NOT NULL CHECK (size > 0),
  replicas INTEGER NOT NULL CHECK (replicas > 0),

  CHECK (used <= size)
);

INSERT INTO storages (name, size, replicas) VALUES ('DUMMY', 100500, 1);

ALTER TABLE volumes
  ADD COLUMN gluster_name TEXT,
  ADD COLUMN storage_id UUID REFERENCES storages (id) ON DELETE CASCADE;

WITH user_volumes AS (
  SELECT DISTINCT owner_user_id, resource_id, resource_label FROM permissions WHERE kind = 'volume'
)
UPDATE volumes
SET gluster_name = CASE deleted
                    WHEN FALSE THEN ENCODE(DIGEST((SELECT resource_label || owner_user_id -- from api-v2
                                  FROM user_volumes
                                  WHERE user_volumes.resource_id = volumes.id), 'sha256'), 'hex')
                    WHEN TRUE THEN '<deleted>'
                    END
WHERE gluster_name IS NULL;
DELETE FROM volumes WHERE gluster_name IS NULL;

CREATE OR REPLACE FUNCTION update_used_in_storage() RETURNS TRIGGER AS $update_used_in_storage$
  BEGIN
    IF NEW.capacity IS NULL THEN
      RETURN NEW;
    END IF;
    CASE TG_OP
      WHEN 'INSERT' THEN
        UPDATE storages SET used = used + NEW.capacity WHERE id = NEW.storage_id;
        RETURN NEW;
      WHEN 'UPDATE' THEN
        IF NEW.deleted = TRUE THEN
          UPDATE storages SET used = used - OLD.capacity WHERE id = NEW.storage_id;
        ELSE
          UPDATE storages SET used = used - OLD.capacity + NEW.capacity WHERE id = NEW.storage_id;
        END IF;
        RETURN NEW;
      WHEN 'DELETE' THEN
        UPDATE storages SET used = used - OLD.capacity WHERE id = NEW.storage_id;
        RETURN OLD;
    END CASE;
    RETURN NEW;
  END;
$update_used_in_storage$ LANGUAGE plpgsql;

CREATE TRIGGER update_used_in_storage BEFORE INSERT OR UPDATE OR DELETE ON volumes
  FOR EACH ROW EXECUTE PROCEDURE update_used_in_storage();

ALTER TABLE volumes DISABLE TRIGGER update_used_in_storage;
WITH dummy_id AS (
  SELECT id FROM storages WHERE name = 'DUMMY'
),
updated_vols AS (
    UPDATE volumes
    SET storage_id = (SELECT id FROM dummy_id)
    WHERE storage_id IS NULL
    RETURNING *
)
UPDATE storages SET used = (SELECT COALESCE(sum(capacity),used) FROM updated_vols WHERE storage_id = storages.id);
ALTER TABLE volumes ENABLE TRIGGER update_used_in_storage;

ALTER TABLE volumes
  ALTER COLUMN gluster_name SET NOT NULL,
  ALTER COLUMN storage_id SET NOT NULL;