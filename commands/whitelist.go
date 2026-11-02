package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/omit"

	"github.com/keyvie/minecrafter/internal/core"
	"github.com/keyvie/minecrafter/internal/database/models"
	"github.com/keyvie/minecrafter/internal/mojang"
	"github.com/keyvie/minecrafter/internal/state"
)

var Whitelist = core.NewApplicationCommand(
	&discord.SlashCommandCreate{
		Name:                     "whitelist",
		Description:              "whitelist command",
		DefaultMemberPermissions: omit.NewPtr(discord.PermissionAdministrator),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommand{
				Name:        "add",
				Description: "Add a player to the whitelist",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:        "username",
						Description: "The minecraft username to add to the whitelist",
						Required:    true,
					},
					discord.ApplicationCommandOptionUser{
						Name:        "user",
						Description: "That user's Discord account",
						Required:    true,
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "remove",
				Description: "Remove a player from the whitelist",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:        "username",
						Description: "The minecraft username to remove from the whitelist",
						Required:    false,
					},
					discord.ApplicationCommandOptionUser{
						Name:        "user",
						Description: "That user's Discord account",
						Required:    false,
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "list",
				Description: "List all players on the whitelist",
			},
		},
	},
	func(
		ctx context.Context,
		c *core.Container[state.Public],
		event *events.ApplicationCommandInteractionCreate,
	) error {
		data := event.SlashCommandInteractionData()

		switch *data.SubCommandName {
		case "add":
			username := data.String("username")
			user := data.User("user")

			// Get the UUID
			profile, err := c.State.Mojang.Profiles.GetByUsername(ctx, username)
			if err != nil {
				return err
			}
			uuid := mojang.FormatUUID(profile.ID)

			// Check if someone already owns that minecraft account?
			exists, err := c.State.Postgres.
				NewSelect().
				Table(models.AccountTable).
				Where("uuid = ?", uuid).
				Exists(ctx)
			if err != nil {
				return err
			}
			if exists {
				return fmt.Errorf("An account with that username has already been linked")
			}

			_, err = c.State.RPC.Call(ctx, "minecraft:allowlist/add", map[string]any{
				"add": []map[string]any{
					{
						"name": username,
						"id":   uuid,
					},
				},
			})
			if err != nil {
				return err
			}

			// Add it to the database
			account := models.Account{
				UUID:   uuid,
				UserID: user.ID.String(),
			}
			_, err = c.State.Postgres.
				NewInsert().
				Model(&account).
				Exec(ctx)
			if err != nil {
				return err
			}

			return event.CreateMessage(discord.MessageCreate{
				Flags:   discord.MessageFlagEphemeral,
				Content: fmt.Sprintf("Added %s to the whitelist", username),
			})

		case "remove":
			username := data.String("username")
			user := data.User("user")

			if username != "" {
				// Get the UUID
				profile, err := c.State.Mojang.Profiles.GetByUsername(ctx, username)
				if err != nil {
					return err
				}
				uuid := mojang.FormatUUID(profile.ID)

				// Check if the account exists in the database
				exists, err := c.State.Postgres.
					NewSelect().
					Table(models.AccountTable).
					Where("uuid = ?", uuid).
					Exists(ctx)
				if err != nil {
					return err
				}
				if !exists {
					return fmt.Errorf("No account with that username is linked")
				}

				_, err = c.State.RPC.Call(ctx, "minecraft:allowlist/remove", map[string]any{
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

				// Remove it from the database
				_, err = c.State.Postgres.
					NewDelete().
					Table(models.AccountTable).
					Where("user_id = ?", user.ID).
					Exec(ctx)

				if err != nil {
					return err
				}

				return event.CreateMessage(discord.MessageCreate{
					Flags:   discord.MessageFlagEphemeral,
					Content: fmt.Sprintf("Removed %s from the whitelist", username),
				})
			}

			if user.ID != 0 {
				// Lookup in database
				var accounts []models.Account
				if err := c.State.Postgres.
					NewSelect().
					Model(&accounts).
					Where("user_id = ?", user.ID).
					Scan(ctx); err != nil {
					return err
				}

				successful := []string{}
				failed := []string{}
				for _, item := range accounts {
					uuid := item.UUID

					// Get Minecraft username
					profile, err := c.State.Mojang.Profiles.GetByUUID(ctx, uuid)
					if err != nil {
						failed = append(failed, uuid)
						continue
					}
					username := profile.Name

					_, err = c.State.RPC.Call(ctx, "minecraft:allowlist/remove", map[string]any{
						"remove": []map[string]any{
							{
								"name": username,
								"id":   uuid,
							},
						},
					})
					if err != nil {
						failed = append(failed, username)
						continue
					}

					_, err = c.State.Postgres.
						NewDelete().
						Table(models.AccountTable).
						Where("uuid = ?", uuid).
						Exec(ctx)
					if err != nil {
						failed = append(failed, username)
						continue
					}
					successful = append(successful, username)
				}

				var responseText []string

				if len(successful) > 0 {
					responseText = append(responseText, fmt.Sprintf(
						"**Successful**\n%s",
						strings.Join(successful, "\n"),
					))
				}

				if len(failed) > 0 {
					responseText = append(responseText, fmt.Sprintf(
						"**Failed**\n%s",
						strings.Join(failed, "\n"),
					))
				}

				return event.CreateMessage(discord.MessageCreate{
					Flags:   discord.MessageFlagEphemeral,
					Content: strings.Join(responseText, "\n\n"),
				})
			}

			return fmt.Errorf("You need to select a user or username")

		case "list":
			resp, err := c.State.RPC.Call(ctx, "minecraft:allowlist")
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
