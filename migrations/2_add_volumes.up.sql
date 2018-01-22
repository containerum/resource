BEGIN TRANSACTION;
CREATE TABLE IF NOT EXISTS volumes (
	id uuid NOT NULL,
	create_time timestamp NOT NULL DEFAULT statement_timestamp(),
	tariff_id uuid NOT NULL,
	deleted boolean NOT NULL DEFAULT false,
	delete_time timestamp NULL,
	capacity int NOT NULL,
	replicas int NOT NULL,

	UNIQUE (id)
);
COMMIT TRANSACTION;
