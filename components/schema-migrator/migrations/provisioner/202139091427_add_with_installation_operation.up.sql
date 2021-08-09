BEGIN;
ALTER TABLE operation ADD COLUMN with_installation bool;
UPDATE operation SET with_installation = true WHERE type = 'PROVISION' AND with_installation IS NULL;
COMMIT;