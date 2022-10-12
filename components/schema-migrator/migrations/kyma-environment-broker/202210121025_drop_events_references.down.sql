ALTER TABLE events
    ADD CONSTRAINT events_instance_id_fkey FOREIGN KEY (instance_id) references instances (instance_id);

ALTER TABLE events
    ADD CONSTRAINT events_operation_id_fkey FOREIGN KEY (operation_id) references operations (id);