BEGIN;

DO $$ BEGIN
    CREATE TYPE event_level AS ENUM ('info', 'error');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS events (
    id             varchar(255) NOT NULL PRIMARY KEY,
    level          event_level NOT NULL,
    instance_id    varchar(255) references instances (instance_id),
    operation_id   varchar(255) references operations (id),
    message        text NOT NULL,
    created_at     timestamp with time zone NOT NULL
);

CREATE INDEX IF NOT EXISTS events_id ON events USING HASH (id);
CREATE INDEX IF NOT EXISTS events_operation_id ON events USING HASH (operation_id);

COMMIT;
