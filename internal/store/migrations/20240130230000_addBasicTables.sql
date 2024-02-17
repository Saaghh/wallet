-- +migrate Up

CREATE TABLE users (
    id uuid not null unique primary key,
    email varchar not null unique,
    registered_at timestamp with time zone default now()
);

CREATE TABLE wallets (
    id uuid not null unique primary key,
    owner_id uuid not null references users(id),
    currency varchar not null,
    balance numeric not null default 0 CHECK ( balance >= 0 ),
    created_at timestamp with time zone default now(),
    modified_at timestamp with time zone default now(),
    name varchar,
    is_disabled boolean not null default false
);

CREATE INDEX idx_wallets_owner_id ON wallets (owner_id);
CREATE INDEX idx_wallets_is_disabled_id ON wallets (is_disabled, id);
CREATE INDEX idx_wallets_modified_at_balance ON wallets (modified_at, balance);
CREATE INDEX idx_wallets_is_disabled_owner_id ON wallets (is_disabled, owner_id);

CREATE TABLE transactions
(
    id             uuid not null unique primary key,
    created_at     timestamp with time zone default now(),
    from_wallet_id uuid references wallets (id),
    to_wallet_id   uuid references wallets (id),
    currency       varchar not null,
    balance        numeric not null
);

CREATE INDEX idx_transactions_from_wallet_id ON transactions (from_wallet_id);
CREATE INDEX idx_transactions_to_wallet_id ON transactions (to_wallet_id);

-- +migrate Down

DROP TABLE users, wallets, transactions CASCADE;