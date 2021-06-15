ALTER TABLE kyma_config ADD COLUMN installer varchar(256) NOT NULL DEFAULT 'KymaOperator';
ALTER TABLE kyma_component_config ADD COLUMN prerequisite boolean NOT NULL DEFAULT false;