BEGIN;

ALTER TABLE gardener_config ADD COLUMN eu_access boolean NOT NULL DEFAULT false;
ALTER TABLE gardener_config ALTER COLUMN eu_access DROP DEFAULT;

COMMIT;
