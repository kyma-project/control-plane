CREATE INDEX operations_by_instance_id ON operations USING btree (instance_id);
CREATE INDEX operations_by_instance_id_date ON operations USING btree (instance_id, created_at);
CREATE INDEX operations_by_orchestration_id ON operations USING btree (orchestration_id);
