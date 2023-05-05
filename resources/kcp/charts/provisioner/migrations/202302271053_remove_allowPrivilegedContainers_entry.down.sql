BEGIN;
ALTER TABLE gardener_config ADD COLUMN allow_privileged_containers boolean NOT NULL DEFAULT true;
COMMIT;