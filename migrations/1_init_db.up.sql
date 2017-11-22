SELECT uuid_generate_v4() as check_uuid_extension;

BEGIN TRANSACTION;
CREATE TABLE namespaces (
	id uuid NOT NULL,
	create_time timestamp NOT NULL DEFAULT statement_timestamp(),
	ram int NOT NULL,
	cpu int NOT NULL,
	max_ext_svc int NOT NULL,
	max_int_svc int NOT NULL,
	max_traffic int NOT NULL,
	deleted boolean NOT NULL DEFAULT false,
	delete_time timestamp NULL,
	tariff_id uuid NOT NULL,

	UNIQUE (id)
);

CREATE TYPE ResourceKind AS ENUM (
	'Namespace',
	'Volume',
	'ExtService',
	'IntService',
	'Domain'
);

CREATE TYPE AccessLevel AS ENUM (
	'none',
	'owner',
	'read',
	'write',
	'readdelete'
);

CREATE TABLE accesses (
	id uuid NOT NULL,
	create_time timestamp NOT NULL DEFAULT statement_timestamp(),
	kind ResourceKind NOT NULL,
	resource_id uuid NOT NULL,
	resource_label varchar NOT NULL,
	user_id uuid NOT NULL,
	owner_user_id uuid NOT NULL,
	access_level AccessLevel NOT NULL,
	limited boolean,
	access_level_change_time timestamp NOT NULL DEFAULT statement_timestamp(),

	UNIQUE (resource_id, user_id),
	UNIQUE (user_id, kind, resource_label),
	CHECK ( user_id = owner_user_id AND limited IS NOT NULL OR user_id <> owner_user_id ),
	CHECK ( user_id != owner_user_id AND limited IS NULL OR user_id = owner_user_id )
);

CREATE TABLE log (
	t timestamp NOT NULL DEFAULT statement_timestamp(),
	action varchar NOT NULL,
	obj_type varchar NOT NULL,
	obj_id varchar NOT NULL
);
COMMIT TRANSACTION;
