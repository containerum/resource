ALTER TABLE deployments
  ALTER COLUMN create_time SET DEFAULT now();

ALTER TABLE ingresses
  ALTER COLUMN created_at SET DEFAULT now();

ALTER TABLE namespaces
  ALTER COLUMN create_time SET DEFAULT now();

ALTER TABLE permissions
  ALTER COLUMN create_time SET DEFAULT now(),
  ALTER COLUMN access_level_change_time SET DEFAULT now();

ALTER TABLE services
  ALTER COLUMN created_at SET DEFAULT now();

ALTER TABLE volumes
  ALTER COLUMN create_time SET DEFAULT now();
