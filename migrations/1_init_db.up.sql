CREATE TABLE resources (
	resource varchar UNIQUE PRIMARY KEY,
	resource_type varchar NOT NULL,
	tariff_id varchar NOT NULL
);

CREATE TABLE namespaces (
	namespace_id varchar UNIQUE PRIMARY KEY,
	resource_id varchar NOT NULL REFERENCES resources,
	namespace_label varchar NOT NULL,
	cpu int NOT NULL,
	memory int NOT NULL
);

CREATE TABLE volumes (
	volume_id varchar UNIQUE PRIMARY KEY,
	resource_id varchar NOT NULL REFERENCES resources,
	volume_label varchar NOT NULL,
	size int NOT NULL
);

CREATE TABLE accesses (
	user_id varchar NOT NULL,
	resource_id varchar REFERENCES resources,
	access varchar NOT NULL
);

CREATE TABLE log (
	t timestamp NOT NULL DEFAULT statement_timestamp(),
	action varchar NOT NULL,
	obj_type varchar NOT NULL,
	obj_id varchar NOT NULL
);
