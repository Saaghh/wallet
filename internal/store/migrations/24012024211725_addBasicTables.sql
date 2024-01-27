-- +migrate Up

CREATE TABLE users (
    id bigserial not null primary key,
    email varchar not null unique,
    regDate date
);

CREATE TABLE wallets (
    id bigserial not null primary key,
    ownerID bigserial not null,
    currency varchar not null,
    balanceFull integer,
    balancePartial integer,
    createdDate date,
    modifiedDate date
);

-- +migrate Down

DROP TABLE users;
DROP TABLE wallets;