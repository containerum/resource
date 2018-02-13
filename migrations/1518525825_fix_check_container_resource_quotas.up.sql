CREATE OR REPLACE FUNCTION check_container_resource_quotas() RETURNS TRIGGER AS $check_container_resource_quotas$
DECLARE
  total_cpu INTEGER;
  total_ram INTEGER;
  limit_ram INTEGER;
  limit_cpu INTEGER;
  container_ns_id UUID;
  msg TEXT = '';
BEGIN
  CASE TG_OP
    WHEN 'INSERT' THEN
    SELECT ns.cpu, ns.ram, ns.id
    INTO limit_cpu, limit_ram, container_ns_id
    FROM deployments d
      JOIN namespaces ns ON d.ns_id = ns.id
    WHERE d.id = NEW.depl_id;

    SELECT sum(c.cpu), sum(c.ram)
    INTO total_cpu, total_ram
    FROM containers c
      JOIN deployments d ON c.depl_id = d.id
    WHERE d.ns_id = container_ns_id;

    total_cpu = total_cpu + NEW.cpu;
    total_ram = total_ram + NEW.ram;
    WHEN 'UPDATE' THEN
    SELECT ns.cpu, ns.ram, ns.id
    INTO limit_cpu, limit_ram, container_ns_id
    FROM deployments d
      JOIN namespaces ns ON d.ns_id = ns.id
    WHERE d.id = OLD.depl_id;

    SELECT sum(c.cpu), sum(c.ram)
    INTO total_cpu, total_ram
    FROM containers c
      JOIN deployments d ON c.depl_id = d.id
    WHERE d.ns_id = container_ns_id;

    IF NEW.ram IS NOT NULL THEN
      total_ram = total_ram - OLD.ram + NEW.ram;
    END IF;
    IF NEW.cpu IS NOT NULL THEN
      total_cpu = total_cpu - OLD.cpu + NEW.cpu;
    END IF;
  END CASE;

  IF total_cpu NOT BETWEEN 0 AND limit_cpu THEN
    msg = msg || FORMAT('cpu %s, ', abs(limit_cpu - total_cpu));
  END IF;
  IF total_ram NOT BETWEEN 0 AND limit_ram THEN
    msg = msg || FORMAT('ram %s, ', abs(limit_ram - total_ram));
  END IF;

  IF msg != '' THEN
    RAISE EXCEPTION 'exceeded namespace resources %', msg;
  END IF;

  RETURN NEW;
END;
$check_container_resource_quotas$ LANGUAGE plpgsql;