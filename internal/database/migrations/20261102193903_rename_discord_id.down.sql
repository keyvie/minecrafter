SET statement_timeout = 0;

--bun:split

ALTER TABLE accounts RENAME COLUMN user_id TO discord_id;