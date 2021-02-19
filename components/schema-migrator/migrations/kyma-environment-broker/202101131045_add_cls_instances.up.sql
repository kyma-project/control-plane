CREATE TABLE IF NOT EXISTS cls_instances (
    id varchar(255) PRIMARY KEY,
    version integer NOT NULL,
    global_account_id varchar(255) NOT NULL,
    region varchar(12) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    removed_by_skr_instance_id varchar(255),
    UNIQUE (global_account_id, removed_by_skr_instance_id);

CREATE TABLE IF NOT EXISTS cls_instance_references (
    id SERIAL,
    cls_instance_id varchar(255) NOT NULL,
    skr_instance_id varchar(255) NOT NULL,
    FOREIGN KEY(cls_instance_id) REFERENCES cls_instances(id) ON DELETE CASCADE;
