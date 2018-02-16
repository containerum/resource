DROP TRIGGER IF EXISTS update_used_in_storage ON volumes;

DROP FUNCTION IF EXISTS update_used_in_storage();

ALTER TABLE volumes
  DROP COLUMN gluster_name,
  DROP COLUMN storage_id;

DROP TABLE IF EXISTS storages;

DROP EXTENSION IF EXISTS pgcrypto;