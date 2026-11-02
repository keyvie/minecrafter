package database

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"

	"github.com/keyvie/minecrafter/internal/database/migrations"
)

func Migrate(ctx context.Context, postgres *bun.DB, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: %s <up|create> [name]", args[0])
	}

	migrator := migrate.NewMigrator(postgres, migrations.Migrations)

	switch args[1] {
	case "up":
		if err := migrator.Init(ctx); err != nil {
			return err
		}

		_, err := migrator.Migrate(ctx)
		if err != nil {
			return err
		}
		slog.Info("Migrate: Successful")
		return nil

	case "create":
		if len(args) < 3 {
			return fmt.Errorf("usage: %s create <name>", args[0])
		}

		_, err := migrator.CreateSQLMigrations(ctx, args[2])
		return err

	default:
		return fmt.Errorf("'%s' is not a valid command", args[1])
	}
}
