CREATE TABLE cluster_administrator
(
    id uuid PRIMARY KEY CHECK (id <> '00000000-0000-0000-0000-000000000000'),
    cluster_id uuid NOT NULL,
    administrator text NOT NULL,
    foreign key (cluster_id) REFERENCES cluster (id) ON DELETE CASCADE
);
