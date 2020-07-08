BEGIN;

ALTER TABLE gardener_config DROP COLUMN enable_kubernetes_version_auto_update;

ALTER TABLE gardener_config DROP COLUMN enable_machine_image_version_auto_update;

COMMIT;
