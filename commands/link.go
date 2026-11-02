package commands

import (
	"context"
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/omit"

	"github.com/keyvie/minecrafter/internal/core"
	"github.com/keyvie/minecrafter/internal/database/models"
	"github.com/keyvie/minecrafter/internal/mojang"
	"github.com/keyvie/minecrafter/internal/state"
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
		ctx context.Context,
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
		exists, err := c.State.Postgres.
			NewSelect().
			Table("accounts").
			Where("uuid = ?", uuid).
			Exists(ctx)
		if err != nil {
			return err
		}
		if exists {
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
		account := models.Account{
			UUID:   uuid,
			UserID: event.User().ID.String(),
		}
		_, err = c.State.Postgres.
			NewInsert().
			Model(&account).
			Exec(ctx)
		if err != nil {
			return err
		}

		return event.CreateMessage(discord.MessageCreate{
			Content: fmt.Sprintf("You have successfully linked %s to your Discord account", username),
		})
	},
)
