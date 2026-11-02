package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/omit"

	"github.com/keyvie/minecrafter/internal/core"
	"github.com/keyvie/minecrafter/internal/rpc"
	"github.com/keyvie/minecrafter/internal/state"
)

var Online = core.NewApplicationCommand(
	&discord.SlashCommandCreate{
		Name:                     "online",
		Description:              "Show online players",
		DefaultMemberPermissions: omit.NewPtr(discord.PermissionsNone),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionBool{
				Name:        "uuids",
				Description: "Incide the UUIDs of the players",
				Required:    false,
			},
		},
	},
	func(
		ctx context.Context,
		c *core.Container[state.Public],
		event *events.ApplicationCommandInteractionCreate,
	) error {
		data := event.SlashCommandInteractionData()
		includeUUIDs := data.Bool("uuids")

		resp, err := c.State.RPC.Call(ctx, "minecraft:players")
		if err != nil {
			return err
		}

		var players []rpc.Player
		if err := json.Unmarshal(resp.Result, &players); err != nil {
			return fmt.Errorf("invalid response: %w", err)
		}

		responseText := []string{"**Online Players**"}
		for _, player := range players {
			if includeUUIDs {
				responseText = append(responseText, fmt.Sprintf("> **%s** (`%s`)", player.Name, player.ID))
			} else {
				responseText = append(responseText, fmt.Sprintf("> **%s**", player.Name))
			}
		}

		if err := event.CreateMessage(discord.MessageCreate{
			Content: strings.Join(responseText, "\n"),
		}); err != nil {
			return err
		}
		return nil
	},
)
