package commands

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/omit"

	"github.com/keyvie/minecrafter/internal/core"
	"github.com/keyvie/minecrafter/internal/database/models"
	"github.com/keyvie/minecrafter/internal/state"
)

var Settings = core.NewApplicationCommand(
	&discord.SlashCommandCreate{
		Name:                     "settings",
		Description:              "Update settings",
		DefaultMemberPermissions: omit.NewPtr(discord.PermissionAdministrator),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommandGroup{
				Name:        "logging",
				Description: "logging",
				Options: []discord.ApplicationCommandOptionSubCommand{
					{
						Name:        "whitelist_channel",
						Description: "The channel for whitelist logging",
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionString{
								Name:        "action",
								Description: "What action to take",
								Required:    true,
								Choices: []discord.ApplicationCommandOptionChoiceString{
									{Name: "Get", Value: "get"},
									{Name: "Set", Value: "set"},
									{Name: "Remove", Value: "remove"},
								},
							},
							discord.ApplicationCommandOptionChannel{
								Name:        "channel",
								Description: "The channel for whitelist logging",
								Required:    false,
							},
						},
					},
					{
						Name:        "connection_channel",
						Description: "The channel for whitelist logging",
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionString{
								Name:        "action",
								Description: "What action to take",
								Required:    true,
								Choices: []discord.ApplicationCommandOptionChoiceString{
									{Name: "Get", Value: "get"},
									{Name: "Set", Value: "set"},
									{Name: "Remove", Value: "remove"},
								},
							},
							discord.ApplicationCommandOptionChannel{
								Name:        "channel",
								Description: "The channel for whitelist logging",
								Required:    false,
							},
						},
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "whitelist_role",
				Description: "whitelist role",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:        "action",
						Description: "What action to take",
						Required:    true,
						Choices: []discord.ApplicationCommandOptionChoiceString{
							{Name: "Get", Value: "get"},
							{Name: "Set", Value: "set"},
							{Name: "Remove", Value: "remove"},
						},
					},
					discord.ApplicationCommandOptionRole{
						Name:        "role",
						Description: "The role to give to whitelisted users",
						Required:    false,
					},
				},
			},
		},
	},
	func(
		ctx context.Context,
		c *core.Container[state.Public],
		event *events.ApplicationCommandInteractionCreate,
	) error {
		data := event.SlashCommandInteractionData()

		guildID := event.GuildID()
		if guildID == nil {
			return fmt.Errorf("command can only be used in a server")
		}

		setting := data.SubCommandName
		if setting == nil {
			return fmt.Errorf("missing setting")
		}

		var config models.Config
		err := c.State.Postgres.NewSelect().
			Model(&config).
			Where("guild_id = ?", guildID.String()).
			Scan(ctx)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("failed to fetch settings: %w", err)
			}
			config = models.Config{
				GuildID:  guildID.String(),
				Settings: models.Settings{},
			}
		}

		var target **string
		switch *setting {
		case "whitelist_channel":
			target = &config.Settings.Logging.WhitelistChannel
		case "connection_channel":
			target = &config.Settings.Logging.ConnectionChannel
		case "whitelist_role":
			target = &config.Settings.WhitelistRole
		}

		action, ok := data.OptString("action")
		if !ok {
			return fmt.Errorf("missing action")
		}

		var content string

		switch action {
		case "get":
			if *target != nil {
				switch strings.Split(*setting, "_")[1] {
				case "channel":
					content = fmt.Sprintf("<#%s>", **target)
				case "role":
					content = fmt.Sprintf("<@&%s>", **target)
				default:
					content = **target
				}
			} else {
				content = "This setting is not configured"
			}
		case "set":
			var id string
			if channel, ok := data.OptChannel("channel"); ok {
				id = channel.ID.String()
			} else if role, ok := data.OptRole("role"); ok {
				id = role.ID.String()
			} else {
				return fmt.Errorf("missing role")
			}
			*target = &id
			content = "Setting updated successfully"
		case "remove":
			*target = nil
			content = "Setting removed successfully"
		}

		if action != "get" {
			_, err := c.State.Postgres.NewInsert().
				Model(&config).
				On("CONFLICT (guild_id) DO UPDATE").
				Set("settings = EXCLUDED.settings").
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to update settings: %w", err)
			}
		}

		return event.CreateMessage(discord.MessageCreate{
			Flags:   discord.MessageFlagEphemeral,
			Content: content,
		})
	},
)
