ALTER TABLE operations
    ADD COLUMN error_msg text,
    ADD COLUMN error_reason text,
    ADD COLUMN error_component text;