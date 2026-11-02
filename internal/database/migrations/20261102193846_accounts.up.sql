SET statement_timeout = 0;

--bun:split

CREATE TABLE IF NOT EXISTS accounts (
    "uuid" TEXT PRIMARY KEY,
    "discord_id" TEXT NOT NULL,
    "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
