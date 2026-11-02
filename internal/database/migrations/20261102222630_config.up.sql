SET statement_timeout = 0;

--bun:split

CREATE TABLE IF NOT EXISTS config (
    "guild_id" TEXT PRIMARY KEY,
    "settings" JSONB NULL
);
