-- +migrate Up

CREATE TABLE transactions (
    id bigserial not null primary key,
    created_at timestamp with time zone default now(),
    from_wallet_id bigint references wallets(id),
    to_wallet_id bigint references wallets(id),
    currency varchar not null,
    balance numeric not null
);

-- +migrate Down

DROP TABLE  transactions;
