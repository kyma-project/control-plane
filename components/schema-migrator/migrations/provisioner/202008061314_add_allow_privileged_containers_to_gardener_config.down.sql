BEGIN;

ALTER TABLE gardener_config DROP COLUMN allow_privileged_containers;

COMMIT;
