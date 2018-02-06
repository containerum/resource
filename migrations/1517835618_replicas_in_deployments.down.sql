ALTER TABLE deployments
  DROP CONSTRAINT replicas_check,
  DROP COLUMN replicas;