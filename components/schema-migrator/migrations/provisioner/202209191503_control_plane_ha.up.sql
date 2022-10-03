BEGIN;
ALTER TABLE gardener_config ADD COLUMN control_plane_failure_tolerance varchar(256);
COMMIT;
