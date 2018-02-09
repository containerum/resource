ALTER TABLE volumes
  ADD COLUMN is_persistent BOOLEAN NOT NULL;

UPDATE volumes SET is_persistent = ns_id IS NOT NULL;

ALTER TABLE volumes
  DROP COLUMN ns_id;
