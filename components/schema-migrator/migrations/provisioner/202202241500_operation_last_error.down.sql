BEGIN;

ALTER TABLE operation DROP COLUMN error_msg text;
ALTER TABLE operation DROP COLUMN error_reason text;
ALTER TABLE operation DROP COLUMN error_component text;

COMMIT;