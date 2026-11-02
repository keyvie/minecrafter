SET statement_timeout = 0;

--bun:split

ALTER TABLE accounts RENAME COLUMN discord_id TO user_id;