BEGIN;
UPDATE gardener_config SET disk_type = '' WHERE disk_type IS NULL;
UPDATE gardener_config SET volume_size_gb = 0 WHERE volume_size_gb IS NULL;
ALTER TABLE gardener_config ALTER COLUMN disk_type SET NOT NULL;
ALTER TABLE gardener_config ALTER COLUMN volume_size_gb SET NOT NULL;
COMMIT;