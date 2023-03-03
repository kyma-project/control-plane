BEGIN;
ALTER TABLE kyma_release DROP COLUMN tiller_yaml;
COMMIT;
