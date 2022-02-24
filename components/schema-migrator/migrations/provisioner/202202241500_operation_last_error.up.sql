BEGIN;

ALTER TABLE operation ADD COLUMN error_msg text;
ALTER TABLE operation ADD COLUMN error_reason text;
ALTER TABLE operation ADD COLUMN error_component text;

COMMIT;