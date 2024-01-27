-- +migrate Up

CREATE TABLE users (
    id bigserial not null primary key,
    email varchar not null unique,
    regDate date
);

CREATE TABLE wallets (
    id bigserial not null primary key,
    ownerID bigserial not null references users(id),
    currency varchar not null,
    balance numeric,
    createdDate date,
    modifiedDate date
);

-- +migrate Down

DROP TABLE users;
DROP TABLE wallets;