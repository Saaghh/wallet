-- +migrate Up

CREATE TABLE users (
    id bigserial not null primary key,
    email varchar not null unique,
    registered_at timestamp with time zone default now()
);

CREATE TABLE wallets (
    id bigserial not null primary key,
    owner_id bigint not null references users(id),
    currency varchar not null,
    balance numeric not null default 0,
    created_at timestamp with time zone default now(),
    modified_at timestamp with time zone default now()
);

-- +migrate Down

DROP TABLE users, wallets CASCADE;