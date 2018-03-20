ALTER TABLE deployments
  ALTER COLUMN create_time SET DEFAULT (now() AT TIME ZONE 'UTC');

ALTER TABLE ingresses
  ALTER COLUMN created_at SET DEFAULT (now() AT TIME ZONE 'UTC');

ALTER TABLE namespaces
  ALTER COLUMN create_time SET DEFAULT (now() AT TIME ZONE 'UTC');

ALTER TABLE permissions
  ALTER COLUMN create_time SET DEFAULT (now() AT TIME ZONE 'UTC'),
  ALTER COLUMN access_level_change_time SET DEFAULT (now() AT TIME ZONE 'UTC');

ALTER TABLE services
  ALTER COLUMN created_at SET DEFAULT (now() AT TIME ZONE 'UTC');

ALTER TABLE volumes
  ALTER COLUMN create_time SET DEFAULT (now() AT TIME ZONE 'UTC');
