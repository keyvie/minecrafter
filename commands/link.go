package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/omit"

	"github.com/keyvie/minecraft-bot/internal/core"
	"github.com/keyvie/minecraft-bot/internal/database"
	"github.com/keyvie/minecraft-bot/internal/database/models"
	"github.com/keyvie/minecraft-bot/internal/mojang"
	"github.com/keyvie/minecraft-bot/internal/state"
)

var Link = core.NewApplicationCommand(
	&discord.SlashCommandCreate{
		Name:                     "link",
		Description:              "link your Minecraft account to your Discord account",
		DefaultMemberPermissions: omit.NewPtr(discord.PermissionAdministrator),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:        "username",
				Description: "Your Minecraft username",
				Required:    true,
			},
		},
	},
	func(
		c *core.Container[state.Public],
		event *events.ApplicationCommandInteractionCreate,
	) error {
		data := event.SlashCommandInteractionData()
		username := data.String("username")

		// Get the UUID
		profile, err := c.State.Mojang.Profiles.GetByUsername(context.Background(), username)
		if err != nil {
			return err
		}
		uuid := mojang.FormatUUID(profile.ID)

		// Check if someone already owns that minecraft account?
		result, err := c.State.Repos.Accounts.FindOne(context.Background(), database.QueryFindOneOptions{
			Where: map[string]any{
				"uuid": uuid,
			},
		})
		if err != nil {
			return err
		}
		if result.Found {
			return fmt.Errorf("An account with that username has already been linked")
		}

		_, err = c.State.RPC.Call(context.Background(), "minecraft:allowlist/add", map[string]any{
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
		_, err = c.State.Repos.Accounts.Create(context.Background(), models.Account{
			UUID:      uuid,
			DiscordID: event.User().ID.String(),
			CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			return err
		}

		return event.CreateMessage(discord.MessageCreate{
			Content: fmt.Sprintf("You have successfully linked %s to your Discord account", username),
		})
	},
)
