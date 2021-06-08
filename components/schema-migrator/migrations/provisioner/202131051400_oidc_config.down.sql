BEGIN;

ALTER TABLE gardener_config DROP COLUMN oidc_config_id;

DROP TABLE oidc_config;

COMMIT;


DROP TABLE signing_algorithms;