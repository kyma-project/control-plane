BEGIN;

ALTER TABLE cluster ADD COLUMN is_kubeconfig_encrypted boolean NOT NULL DEFAULT false;
ALTER TABLE cluster ALTER COLUMN is_kubeconfig_encrypted DROP DEFAULT;

ALTER TABLE cluster_administrator ADD COLUMN is_user_id_encrypted boolean NOT NULL DEFAULT false;
ALTER TABLE cluster_administrator ALTER COLUMN is_user_id_encrypted DROP DEFAULT;

COMMIT;
