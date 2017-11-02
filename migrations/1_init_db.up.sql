SELECT uuid_generate_v4() as check_uuid_extension;

CREATE TABLE namespaces (
	id uuid NOT NULL,
	label varchar NOT NULL,
	user_id uuid NOT NULL,
	create_time timestamp NOT NULL DEFAULT statement_timestamp(),
	ram int NOT NULL,
	cpu int NOT NULL,
	max_ext_svc int NOT NULL,
	max_int_svc int NOT NULL,
	max_traffic int NOT NULL,
	deleted boolean NOT NULL DEFAULT false,
	delete_time timestamp NULL,
	tariff_id uuid NOT NULL
);

CREATE TYPE ResourceKind AS ENUM (
	'Namespace',
	'Volume',
	'ExtService',
	'IntService',
	'Domain'
);

CREATE TYPE PermStatus AS ENUM (
	'none',
	'owner',
	'read',
	'write',
	'readdelete'
);

CREATE TABLE permissions (
	id uuid NOT NULL,
	kind ResourceKind NOT NULL,
	resource_id uuid NOT NULL,
	user_id uuid NOT NULL,
	create_time timestamp NOT NULL DEFAULT statement_timestamp(),
	status_main PermStatus NOT NULL,
	limited boolean NOT NULL DEFAULT false,
	status_limited PermStatus NULL,
	status_change_time timestamp NOT NULL
);

CREATE TABLE log (
	t timestamp NOT NULL DEFAULT statement_timestamp(),
	action varchar NOT NULL,
	obj_type varchar NOT NULL,
	obj_id varchar NOT NULL
);
