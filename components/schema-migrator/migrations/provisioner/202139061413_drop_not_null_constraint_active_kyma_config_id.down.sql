BEGIN;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
UPDATE cluster SET active_kyma_config_id = uuid_generate_v1() WHERE active_kyma_config_id IS NULL;
ALTER TABLE cluster ALTER COLUMN active_kyma_config_id SET NOT NULL;
COMMIT;