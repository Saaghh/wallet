-- +migrate Up

CREATE TABLE users (
    id uuid not null unique primary key default gen_random_uuid(),
    email varchar not null unique,
    registered_at timestamp with time zone default now()
);

CREATE TABLE wallets (
    id uuid not null unique primary key default gen_random_uuid(),
    owner_id uuid not null references users(id),
    currency varchar not null,
    balance numeric not null default 0 CHECK ( balance >= 0 ),
    created_at timestamp with time zone default now(),
    modified_at timestamp with time zone default now(),
    name varchar,
    is_disabled boolean not null default false

);

CREATE TABLE transactions
(
    id             uuid not null unique primary key,
    created_at     timestamp with time zone default now(),
    from_wallet_id uuid references wallets (id),
    to_wallet_id   uuid references wallets (id),
    currency       varchar not null,
    balance        numeric not null
);

-- +migrate Down

DROP TABLE users, wallets, transactions CASCADE;