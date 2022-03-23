BEGIN;

ALTER TABLE gardener_config ADD COLUMN shoot_networking_filter_disabled boolean;

COMMIT;