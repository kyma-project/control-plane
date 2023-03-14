BEGIN;
ALTER TABLE kyma_release ADD COLUMN tiller_yaml text NOT NULL;
COMMIT;
