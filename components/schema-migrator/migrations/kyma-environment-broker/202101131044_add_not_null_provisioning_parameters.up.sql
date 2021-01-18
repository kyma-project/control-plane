BEGIN;

ALTER TABLE operations
    ALTER COLUMN provisioning_parameters SET NOT NULL;

COMMIT;
