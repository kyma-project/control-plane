DELETE FROM operation WHERE operation_type = 'UPGRADE_SHOOT';

ALTER TYPE operation_type RENAME TO operation_type_old;

CREATE TYPE operation_type AS ENUM (
    'PROVISION',
    'UPGRADE',
    'DEPROVISION',
    'RECONNECT_RUNTIME'
    );


ALTER TABLE operation ALTER COLUMN type TYPE operation_type USING type::text::operation_type;

DROP TYPE operation_type_old;