CREATE TABLE IF NOT EXISTS orchestrations (
    orchestration_id varchar(255) PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL,
	state varchar(32) NOT NULL,
	parameters text NOT NULL,
	description text,
	runtime_operations text
);
