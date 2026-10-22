package main

import (
	"context"
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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"

	"github.com/keyvie/minecraft-bot/commands"
	"github.com/keyvie/minecraft-bot/internal/core"
	"github.com/keyvie/minecraft-bot/internal/database"
	"github.com/keyvie/minecraft-bot/internal/database/models"
	"github.com/keyvie/minecraft-bot/internal/modules"
	"github.com/keyvie/minecraft-bot/internal/mojang"
	"github.com/keyvie/minecraft-bot/internal/rpc"
	"github.com/keyvie/minecraft-bot/internal/state"
	"github.com/keyvie/minecraft-bot/listeners"
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

	pool, err := pgxpool.New(ctx, postgresUrl)
	if err != nil {
		panic(err)
	}
	if err := pool.Ping(ctx); err != nil {
		panic(err)
	}

	// Check if args passes migrate
	builtInMigrate := strings.ToLower(os.Getenv("MIGRATE"))
	if (len(os.Args) > 1 && os.Args[1] == "migrate") || builtInMigrate == "true" {
		args := os.Args[1:]
		if builtInMigrate == "true" {
			args = []string{"migrate", "up"}
		}
		db := stdlib.OpenDBFromPool(pool)
		if err := database.Migrate(ctx, db, args); err != nil {
			slog.Error("Migrate", "error", err.Error())
		}
		if builtInMigrate != "true" {
			return
		}
	}

	// Create all the repos
	accountsRepo := database.NewRepository[models.Account](pool, "accounts")

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

	state := state.New(rpcClient, mojangClient, state.Repos{
		Accounts: accountsRepo,
	})
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

	pool.Close()
}
