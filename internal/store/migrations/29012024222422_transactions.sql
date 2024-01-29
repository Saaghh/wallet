-- +migrate Up

CREATE TABLE transactions (
    id bigserial not null primary key,
    created_at timestamp with time zone default now(),
    finished_at timestamp with time zone,
    from_wallet_id bigint not null references wallets(id),
    to_wallet_id bigint not null references wallets(id),
    currency varchar not null,
    balance numeric not null
);

CREATE TABLE external_transactions (
    id bigserial not null primary key,
    created_at timestamp with time zone default now(),
    finished_at timestamp with time zone,
    external_agent varchar not null,
    wallet_id bigint references wallets(id),
    currency varchar not null,
    balance numeric not null
);

-- +migrate Down

DROP TABLE  transactions, external_transactions;
