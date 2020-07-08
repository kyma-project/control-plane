BEGIN;

ALTER TABLE gardener_config ADD COLUMN enable_kubernetes_version_auto_update boolean NOT NULL DEFAULT false;
ALTER TABLE gardener_config ALTER COLUMN enable_kubernetes_version_auto_update DROP DEFAULT;

ALTER TABLE gardener_config ADD COLUMN enable_machine_image_version_auto_update boolean NOT NULL DEFAULT false;
ALTER TABLE gardener_config ALTER COLUMN enable_machine_image_version_auto_update DROP DEFAULT;

COMMIT;
