BEGIN;

ALTER TABLE gardener_config DROP COLUMN shoot_networking_filter_disabled;

COMMIT;
