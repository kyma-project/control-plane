BEGIN;
ALTER TABLE gardener_config DROP COLUMN pods_cidr;
ALTER TABLE gardener_config DROP COLUMN services_cidr;
COMMIT;