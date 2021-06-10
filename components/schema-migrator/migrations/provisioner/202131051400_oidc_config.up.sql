CREATE TABLE oidc_config
(
    id uuid PRIMARY KEY CHECK (id <> '00000000-0000-0000-0000-000000000000'),
    gardener_config_id uuid NOT NULL,
    client_id text NOT NULL,
    groups_claim text NOT NULL,
    issuer_url text NOT NULL,
    username_claim text NOT NULL,
    username_prefix text NOT NULL,
    foreign key (gardener_config_id) REFERENCES gardener_config (id) ON DELETE CASCADE
);

CREATE TABLE signing_algorithms
(
    id uuid PRIMARY KEY CHECK (id <> '00000000-0000-0000-0000-000000000000'),
    oidc_config_id uuid NOT NULL,
    algorithm text NOT NULL,
    foreign key (oidc_config_id) REFERENCES oidc_config (id) ON DELETE CASCADE
);