package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/omit"

	"github.com/keyvie/minecraft-bot/internal/core"
	"github.com/keyvie/minecraft-bot/internal/mojang"
	"github.com/keyvie/minecraft-bot/internal/state"
)

var Operator = core.NewApplicationCommand(
	&discord.SlashCommandCreate{
		Name:                     "operator",
		Description:              "op command",
		DefaultMemberPermissions: omit.NewPtr(discord.PermissionAdministrator),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommand{
				Name:        "add",
				Description: "Add a player as an operator",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:        "username",
						Description: "The minecraft username to give operator",
						Required:    true,
					},
					discord.ApplicationCommandOptionInt{
						Name:        "level",
						Description: "The operator level to assign",
						Required:    false,
					},
					discord.ApplicationCommandOptionBool{
						Name:        "bypass",
						Description: "If the player bypasses the player limit",
						Required:    false,
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "remove",
				Description: "Remove a player from operators",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:        "username",
						Description: "The minecraft username to remove from operators",
						Required:    true,
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "list",
				Description: "List all players with operator status",
			},
		},
	},
	func(
		c *core.Container[state.Public],
		event *events.ApplicationCommandInteractionCreate,
	) error {
		data := event.SlashCommandInteractionData()

		switch *data.SubCommandName {
		case "add":
			username := data.String("username")
			level, levelExits := data.OptInt("level")
			if !levelExits {
				level = 1
			}
			bypass, bypassExits := data.OptBool("bypass")
			if !bypassExits {
				bypass = false
			}

			// Get the UUID
			profile, err := c.State.Mojang.Profiles.GetByUsername(context.Background(), username)
			if err != nil {
				return err
			}
			uuid := mojang.FormatUUID(profile.ID)

			_, err = c.State.RPC.Call(context.Background(), "minecraft:operators/add", map[string]any{
				"add": []map[string]any{
					{
						"permissionLevel":     level,
						"bypassesPlayerLimit": bypass,
						"player": map[string]any{
							"id":   uuid,
							"name": username,
						},
					},
				},
			})
			if err != nil {
				return err
			}

			return event.CreateMessage(discord.MessageCreate{
				Flags:   discord.MessageFlagEphemeral,
				Content: fmt.Sprintf("Added %s as an operator", username),
			})

		case "remove":
			username := data.String("username")

			// Get the UUID
			profile, err := c.State.Mojang.Profiles.GetByUsername(context.Background(), username)
			if err != nil {
				return err
			}
			uuid := mojang.FormatUUID(profile.ID)

			_, err = c.State.RPC.Call(context.Background(), "minecraft:operators/remove", map[string]any{
				"remove": []map[string]any{
					{
						"name": username,
						"id":   uuid,
					},
				},
			})
			if err != nil {
				return err
			}

			return event.CreateMessage(discord.MessageCreate{
				Flags:   discord.MessageFlagEphemeral,
				Content: fmt.Sprintf("Removed %s from operators", username),
			})

		case "list":
			resp, err := c.State.RPC.Call(context.Background(), "minecraft:operators")
			if err != nil {
				return err
			}

			var pretty bytes.Buffer
			if err := json.Indent(&pretty, resp.Result, "", "  "); err != nil {
				return fmt.Errorf("failed to format RPC response: %w", err)
			}

			return event.CreateMessage(discord.MessageCreate{
				Flags: discord.MessageFlagEphemeral,
				Files: []*discord.File{
					{
						Name:   "response.json",
						Reader: bytes.NewReader(pretty.Bytes()),
					},
				},
			})
		}
		return fmt.Errorf("unknown subcommand: %s", *data.SubCommandName)
	},
)
