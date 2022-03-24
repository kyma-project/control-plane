CREATE TABLE dns_config
(
    id uuid PRIMARY KEY CHECK (id <> '00000000-0000-0000-0000-000000000000'),
    domain varchar(256) NOT NULL,
    gardener_config_id uuid NOT NULL,
    unique(gardener_config_id),
    foreign key (gardener_config_id) REFERENCES gardener_config (id) ON DELETE CASCADE
);


CREATE TABLE dns_providers
(
    id uuid PRIMARY KEY CHECK (id <> '00000000-0000-0000-0000-000000000000'),
    dns_config_id uuid NOT NULL,
    domains_include varchar(256) NOT NULL,
    is_primary boolean NOT NULL,
    secret_name varchar(256) NOT NULL,
    type varchar(256) NOT NULL,
    foreign key (dns_config_id) REFERENCES dns_config (id) ON DELETE CASCADE
);
