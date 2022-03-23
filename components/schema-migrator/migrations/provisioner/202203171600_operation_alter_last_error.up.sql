ALTER TABLE operation
    ALTER COLUMN err_message SET DEFAULT '',
    ALTER COLUMN reason SET DEFAULT '',
    ALTER COLUMN component SET DEFAULT '';

UPDATE operation SET err_message = '' WHERE err_message IS NULL;
UPDATE operation SET reason = '' WHERE reason IS NULL;
UPDATE operation SET component = '' WHERE component IS NULL;