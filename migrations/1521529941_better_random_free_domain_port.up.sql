DROP FUNCTION IF EXISTS random_free_domain_port(_domain TEXT, _protocol PROTOCOL);

CREATE OR REPLACE FUNCTION random_free_domain_port(_domain TEXT, _lower INTEGER, _upper INTEGER, _protocol PROTOCOL) RETURNS INTEGER AS $random_free_domain_port$
DECLARE
  rand_port INTEGER;
BEGIN
  SELECT generate_series(_lower, _upper) AS ser
  EXCEPT
  SELECT sp.port
  FROM service_ports sp
  JOIN domains d ON sp.domain_id = d.id
  WHERE (d.domain, sp.protocol) = (_domain, _protocol)
  ORDER BY ser ASC
  LIMIT 1
  INTO rand_port;
  RETURN rand_port;
END;
$random_free_domain_port$ LANGUAGE plpgsql;