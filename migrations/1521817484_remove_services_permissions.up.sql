ALTER TYPE RESOURCE_KIND RENAME TO RESOURCE_KIND_OLD;

CREATE TYPE RESOURCE_KIND AS ENUM ('namespace', 'volume');

DELETE FROM permissions WHERE kind IN ('extservice', 'intservice');

ALTER TABLE permissions ALTER COLUMN kind SET DATA TYPE RESOURCE_KIND USING kind::TEXT::RESOURCE_KIND;

DROP TYPE RESOURCE_KIND_OLD;