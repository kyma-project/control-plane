BEGIN;
ALTER TABLE gardener_config ADD COLUMN pods_cidr varchar(256);
ALTER TABLE gardener_config ADD COLUMN services_cidr varchar(256);
COMMIT;