BEGIN TRANSACTION;
CREATE TABLE IF NOT EXISTS deployments (
	id uuid NOT NULL CONSTRAINT prim_key PRIMARY KEY,
	ns_id uuid NOT NULL,
	name varchar NOT NULL,
	ram int NOT NULL,
	cpu int NOT NULL,
	create_time timestamp NOT NULL,
	deleted boolean NOT NULL,
	delete_time timestamp NOT NULL,

	CONSTRAINT uniq_in_ns UNIQUE (ns_id, name),
	CONSTRAINT valid_resources CHECK (ram > 0 AND cpu > 0)
);
CREATE TABLE IF NOT EXISTS deployment_volume (
	depl_id uuid NOT NULL,
	vol_id uuid NOT NULL,

	CONSTRAINT uniq_pair UNIQUE(depl_id, vol_id)
);
CREATE TABLE IF NOT EXISTS containers (
	depl_id uuid NOT NULL,
	image varchar NOT NULL
);
COMMIT TRANSACTION;
