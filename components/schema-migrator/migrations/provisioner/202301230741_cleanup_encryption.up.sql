BEGIN;

ALTER TABLE cluster DROP COLUMN is_kubeconfig_encrypted;

ALTER TABLE cluster_administrator DROP COLUMN is_user_id_encrypted;

COMMIT;
