CREATE OR REPLACE FUNCTION check_container_resource_quotas() RETURNS TRIGGER AS $check_container_resource_quotas$
DECLARE
  totalCPU INTEGER;
  totalRAM INTEGER;
  limitRAM INTEGER;
  limitCPU INTEGER;
  msg TEXT = '';
BEGIN
  CASE TG_OP
    WHEN 'INSERT' THEN
    SELECT cpu, ram INTO limitCPU, limitRAM FROM deployments WHERE id = NEW.depl_id;
    SELECT sum(cpu), sum(ram) deployments INTO totalCPU, totalRAM FROM containers WHERE depl_id = NEW.depl_id;
    totalCPU = totalCPU + NEW.cpu;
    totalRAM = totalRAM + NEW.ram;
    WHEN 'UPDATE' THEN
    SELECT cpu, ram INTO limitCPU, limitRAM FROM deployments WHERE id = OLD.depl_id;
    SELECT sum(cpu), sum(ram) INTO totalCPU, totalRAM FROM deployments WHERE id = OLD.depl_id;
    IF NEW.ram IS NOT NULL THEN
      totalRAM = totalRAM - OLD.ram + NEW.ram;
    END IF;
    IF NEW.cpu IS NOT NULL THEN
      totalCPU = totalCPU - OLD.cpu + NEW.cpu;
    END IF;
  END CASE;

  IF totalCPU NOT BETWEEN 0 AND limitCPU THEN
    msg = msg || FORMAT('cpu %s, ', abs(limitCPU - totalCPU));
  END IF;
  IF totalRAM NOT BETWEEN 0 AND limitRAM THEN
    msg = msg || FORMAT('ram %s, ', abs(limitRAM - totalRAM));
  END IF;

  IF msg != '' THEN
    RAISE EXCEPTION 'exceeded namespace resources %', msg;
  END IF;

  RETURN NEW;
END;
$check_container_resource_quotas$ LANGUAGE plpgsql;

ALTER TABLE deployments
  ADD COLUMN ram INTEGER CHECK (ram > 0),
  ADD COLUMN cpu INTEGER CHECK (cpu > 0)

CREATE OR REPLACE FUNCTION check_deploy_resource_quotas() RETURNS TRIGGER AS $check_deploy_resource_quotas$
DECLARE
  totalCPU INTEGER;
  totalRAM INTEGER;
  limitRAM INTEGER;
  limitCPU INTEGER;
  msg TEXT = '';
BEGIN
  CASE TG_OP
    WHEN 'INSERT' THEN
    SELECT cpu, ram INTO limitCPU, limitRAM FROM namespaces WHERE id = NEW.ns_id;
    SELECT sum(cpu), sum(ram) deployments INTO totalCPU, totalRAM FROM deployments WHERE ns_id = NEW.ns_id;
    totalCPU = totalCPU + NEW.cpu;
    totalRAM = totalRAM + NEW.ram;
    WHEN 'UPDATE' THEN
    SELECT cpu, ram INTO limitCPU, limitRAM FROM namespaces WHERE id = OLD.ns_id;
    SELECT sum(cpu), sum(ram) INTO totalCPU, totalRAM FROM deployments WHERE ns_id = OLD.ns_id;
    IF NEW.ram IS NOT NULL THEN
      totalRAM = totalRAM - OLD.ram + NEW.ram;
    END IF;
    IF NEW.cpu IS NOT NULL THEN
      totalCPU = totalCPU - OLD.cpu + NEW.cpu;
    END IF;
  END CASE;

  IF totalCPU NOT BETWEEN 0 AND limitCPU THEN
    msg = msg || FORMAT('cpu %s, ', abs(limitCPU - totalCPU));
  END IF;
  IF totalRAM NOT BETWEEN 0 AND limitRAM THEN
    msg = msg || FORMAT('ram %s, ', abs(limitRAM - totalRAM));
  END IF;

  IF msg != '' THEN
    RAISE EXCEPTION 'exceeded namespace resources %', msg;
  END IF;

  RETURN NEW;
END;
$check_deploy_resource_quotas$ LANGUAGE plpgsql;

CREATE TRIGGER check_deploy_resource_quotas BEFORE INSERT OR UPDATE ON deployments
  FOR EACH ROW EXECUTE PROCEDURE check_deploy_resource_quotas();
