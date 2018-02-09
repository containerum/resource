DROP TRIGGER IF EXISTS check_unique_for_user_domain ON ingresses;

DROP FUNCTION IF EXISTS check_unique_for_user_domain();

DROP TABLE IF EXISTS ingresses;

DROP TYPE IF EXISTS INGRESS_TYPE;