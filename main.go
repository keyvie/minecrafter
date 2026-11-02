package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/gateway"

	"github.com/joho/godotenv"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/keyvie/minecrafter/commands"
	"github.com/keyvie/minecrafter/internal/core"
	"github.com/keyvie/minecrafter/internal/database"
	"github.com/keyvie/minecrafter/internal/modules"
	"github.com/keyvie/minecrafter/internal/mojang"
	"github.com/keyvie/minecrafter/internal/rpc"
	"github.com/keyvie/minecrafter/internal/state"
	"github.com/keyvie/minecrafter/internal/validator"
	"github.com/keyvie/minecrafter/listeners"
)

func main() {
	// Load env values from .env if it exists
	_ = godotenv.Load()

	// Setup logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if strings.ToLower(os.Getenv("DEBUG")) == "true" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else {
		slog.SetLogLoggerLevel(slog.LevelInfo)
	}

	// Listen for signals to shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Checks for database connection url
	postgresUrl, exists := os.LookupEnv("POSTGRES_URL")
	if !exists || postgresUrl == "" {
		panic("POSTGRES_URL is not set")
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(postgresUrl)))
	postgres := bun.NewDB(sqldb, pgdialect.New())
	if err := postgres.Ping(); err != nil {
		panic(err)
	}

	// Check if args passes migrate
	builtInMigrate := strings.ToLower(os.Getenv("MIGRATE"))
	if (len(os.Args) > 1 && os.Args[1] == "migrate") || builtInMigrate == "true" {
		args := os.Args[1:]
		if builtInMigrate == "true" {
			args = []string{"migrate", "up"}
		}
		if err := database.Migrate(ctx, postgres, args); err != nil {
			slog.Error("Migrate", "error", err.Error())
		}
		if os.Args[1] == "migrate" {
			return
		}
	}

	// Open the discord client
	discordClient, err := disgo.New(os.Getenv("DISCORD_TOKEN"),
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(
				gateway.IntentsAll,
			),
		),
	)
	if err != nil {
		panic(err)
	}
	//defer discordClient.Close(context.Background())

	// Create Monjang API Wrapper
	mojangClient := mojang.NewClient()

	// Create RPC client
	rpcClient := rpc.NewClient(rpc.WithReconnect(true), rpc.WithReconnectTimeout(30))
	var secret string
	if s, ok := os.LookupEnv("RPC_SECRET"); ok && s != "" {
		secret = s
	}
	if err = rpcClient.Connect(ctx, os.Getenv("RPC_URL"), rpc.WithSecret(secret)); err != nil {
		panic(err)
	}

	v := validator.New()

	state := state.New(postgres, rpcClient, mojangClient, v)
	container := core.NewContainer(discordClient, core.WithState(*state))
	core.AddHandlers(container)

	// Add all listeners
	if err = modules.AddListeners(container, listeners.Ready); err != nil {
		panic(err)
	}

	// Add all commands
	if err = modules.AddCommands(container,
		commands.Link,
		commands.Online,
		commands.Operator,
		commands.Ping,
		commands.RPC,
		commands.Settings,
		commands.Whitelist,
	); err != nil {
		panic(err)
	}

	// Open connection to Discord
	if err = discordClient.OpenGateway(ctx); err != nil {
		panic(err)
	}

	// What gets ran after shutdown is triggered
	<-ctx.Done()
	stop()

	// Make a context for the shutdown process
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	discordClient.Close(shutdownCtx)

	if err := rpcClient.Close(shutdownCtx); err != nil {
		slog.Error("RPC", "Error closing", err)
	}

	if err := postgres.Close(); err != nil {
		slog.Error("PostgreSQL", "Error closing", err)
	}
}
