package models

import "github.com/uptrace/bun"

const ConfigTable = "config"

type Config struct {
	bun.BaseModel `bun:"table:config"`

	GuildID  string   `bun:"guild_id,pk"`
	Settings Settings `bun:"settings,type:jsonb"`
}

type Settings struct {
	Logging       SettingsLogging `json:"logging"`
	WhitelistRole *string         `json:"whitelist_role" validate:"required,snowflake"`
}

type SettingsLogging struct {
	WhitelistChannel  *string `json:"whitelist_channel" validate:"required,snowflake"`
	ConnectionChannel *string `json:"connection_channel" validate:"required,snowflake"`
}
