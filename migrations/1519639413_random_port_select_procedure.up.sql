CREATE OR REPLACE FUNCTION random_free_domain_port(_domain TEXT, _protocol PROTOCOL) RETURNS INTEGER AS $random_free_domain_port$
DECLARE
  rand_port INTEGER;
BEGIN
  LOOP
    SELECT floor(random() * (65535 - 11000) + 11000) INTO rand_port;
    EXIT WHEN NOT EXISTS(
        SELECT sp.port
        FROM service_ports sp
          JOIN domains d ON sp.domain_id = d.id
        WHERE (d.domain, sp.protocol) = (_domain, _protocol)
    );
  END LOOP;
  RETURN rand_port;
END;
$random_free_domain_port$ LANGUAGE plpgsql;