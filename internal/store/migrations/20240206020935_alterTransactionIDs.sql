-- +migrate Up

    ALTER TABLE transactions ADD COLUMN uuids uuid not null unique default gen_random_uuid();
    ALTER TABLE transactions DROP COLUMN id;
    ALTER TABLE transactions ADD PRIMARY KEY (uuids);
    ALTER TABLE transactions RENAME COLUMN uuids TO id;

-- +migrate Down
