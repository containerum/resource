BEGIN TRANSACTION;
CREATE TABLE namespaceVolume (
	ns_id uuid NOT NULL,
	vol_id uuid NOT NULL,

	UNIQUE (ns_id, vol_id),
	UNIQUE (vol_id)
);
COMMIT TRANSACTION;
