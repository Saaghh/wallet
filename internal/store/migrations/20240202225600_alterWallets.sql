-- +migrate Up

ALTER TABLE wallets
ADD COLUMN name varchar,
ADD COLUMN is_disabled boolean not null default false;

-- +migrate Down

ALTER TABLE wallets
DROP COLUMN name,
DROP COLUMN is_disabled;