-- +migrate Up

CREATE TABLE users (
    id uuid not null primary key default gen_random_uuid(),
    email varchar not null unique,
    registered_at timestamp with time zone default now()
);

CREATE TABLE wallets (
    id uuid not null primary key default gen_random_uuid(),
    owner_id uuid not null references users(id) on delete cascade ,
    currency varchar not null,
    balance numeric not null default 0,
    created_at timestamp with time zone default now(),
    modified_at timestamp with time zone default now()
);

-- +migrate Down

DROP TABLE users, wallets CASCADE;