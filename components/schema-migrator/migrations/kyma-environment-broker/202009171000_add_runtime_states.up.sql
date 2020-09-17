CREATE TABLE IF NOT EXISTS runtime_states (
    id varchar(255) PRIMARY KEY,
    runtime_id varchar(255),
    operation_id varchar(255),
    created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL,
	kyma_config text,
	cluster_config text,
	kyma_version text,
	k8s_version text
);
