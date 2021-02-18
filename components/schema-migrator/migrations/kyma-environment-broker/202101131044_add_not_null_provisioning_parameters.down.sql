BEGIN;

ALTER TABLE operations
    ALTER COLUMN provisioning_parameters DROP NOT NULL;

COMMIT;
