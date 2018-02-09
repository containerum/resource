CREATE TYPE INGRESS_TYPE AS ENUM (
  'http',
  'https',
  'custom_https'
);

CREATE TABLE IF NOT EXISTS ingresses (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  custom_domain TEXT NOT NULL,
  type INGRESS_TYPE NOT NULL DEFAULT 'http',
  service_id UUID REFERENCES services (id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT (now() AT TIME ZONE 'UTC')
);

CREATE OR REPLACE FUNCTION check_unique_for_user_domain() RETURNS TRIGGER AS $check_unique_for_user_domain$
  DECLARE
    ingress_userid UUID;
  BEGIN
    SELECT p.owner_user_id
    INTO ingress_userid
    FROM services s
    JOIN deployments d ON s.deploy_id = d.id
    JOIN permissions p ON p.resource_id = d.ns_id AND p.kind = 'namespace'
    WHERE s.id = NEW.service_id;

    IF EXISTS(
      SELECT 1
      FROM permissions p
      JOIN namespaces ns ON p.resource_id = ns.id AND p.kind = 'namespace'
      JOIN deployments d ON ns.id = d.ns_id
      JOIN services s ON s.deploy_id = d.id
      JOIN ingresses i ON i.service_id = s.id
      WHERE p.owner_user_id = ingress_userid
    ) THEN
      RAISE EXCEPTION 'ingress domain % already added for user, can`t add', NEW.custom_domain;
    END IF;

    RETURN NEW;
  END;
$check_unique_for_user_domain$ LANGUAGE plpgsql;

CREATE TRIGGER check_unique_for_user_domain BEFORE INSERT OR UPDATE ON ingresses
  FOR EACH ROW EXECUTE PROCEDURE check_unique_for_user_domain();