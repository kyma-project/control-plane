BEGIN;

ALTER TABLE gardener_config ADD COLUMN allow_privileged_containers boolean NOT NULL DEFAULT true;
ALTER TABLE gardener_config ALTER COLUMN allow_privileged_containers DROP DEFAULT;

COMMIT;
