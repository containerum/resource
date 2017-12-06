BEGIN TRANSACTION;
CREATE TABLE namespace_volume (
	ns_id uuid NOT NULL,
	vol_id uuid NOT NULL,

	UNIQUE (ns_id, vol_id),
	UNIQUE (vol_id)
);
ALTER TABLE volumes ADD COLUMN is_persistent bool NOT NULL;
COMMIT TRANSACTION;
