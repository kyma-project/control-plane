CREATE TABLE IF NOT EXISTS cls_instances (
    id varchar(255) PRIMARY KEY,
    global_account_id varchar(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    unique (global_account_id)
);
