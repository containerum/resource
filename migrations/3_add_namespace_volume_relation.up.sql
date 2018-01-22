BEGIN TRANSACTION;
CREATE TABLE IF NOT EXISTS namespace_volume (
	ns_id uuid NOT NULL,
	vol_id uuid NOT NULL,

	UNIQUE (ns_id, vol_id),
	UNIQUE (vol_id)
);
ALTER TABLE volumes
	ADD COLUMN IF NOT EXISTS is_persistent bool NOT NULL;
COMMIT TRANSACTION;
