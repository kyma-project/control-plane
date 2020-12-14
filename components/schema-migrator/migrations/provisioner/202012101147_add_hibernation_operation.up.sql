BEGIN;

ALTER TYPE operation_type ADD VALUE 'HIBERNATE' AFTER 'UPGRADE_SHOOT';

ALTER TABLE gardener_config ADD COLUMN hibernated boolean;

COMMIT;