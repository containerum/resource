ALTER TABLE service_ports ALTER COLUMN port DROP NOT NULL;

UPDATE service_ports SET target_port = "port", "port" = target_port;

ALTER TABLE service_ports ALTER COLUMN target_port SET NOT NULL;