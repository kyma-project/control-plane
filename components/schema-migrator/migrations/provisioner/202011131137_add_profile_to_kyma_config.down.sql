BEGIN;

ALTER TABLE kyma_config DROP COLUMN profile;

DROP TYPE kyma_profile;

COMMIT;
