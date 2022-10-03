BEGIN;
ALTER TABLE gardener_config DROP COLUMN control_plane_failure_tolerance;
COMMIT;
