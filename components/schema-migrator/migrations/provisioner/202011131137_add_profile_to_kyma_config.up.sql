BEGIN;

CREATE TYPE kyma_profile AS ENUM (
    'EVALUATION',
    'PRODUCTION'
);


ALTER TABLE kyma_config ADD COLUMN profile kyma_profile;

COMMIT;
