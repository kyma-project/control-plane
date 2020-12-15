BEGIN;

DELETE FROM operation WHERE type = 'HIBERNATE';

ALTER TYPE operation_type RENAME TO operation_type_old;

CREATE TYPE operation_type AS ENUM (
    'PROVISION',
    'UPGRADE',
    'DEPROVISION',
    'RECONNECT_RUNTIME',
    'UPGRADE_SHOOT'
    );


ALTER TABLE operation ALTER COLUMN type TYPE operation_type USING type::text::operation_type;

DROP TYPE operation_type_old;

COMMIT;