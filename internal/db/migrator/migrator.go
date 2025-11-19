package migrator

import (
	"context"

	"github.com/boratanrikulu/sendpulse/internal/config"
	"github.com/uptrace/bun/migrate"
)

// InitMigrator creates migration tables
func InitMigrator(ctx context.Context, migrator *migrate.Migrator) error {
	return migrator.Init(ctx)
}

// Migrate runs all migrations.
func Migrate(ctx context.Context, migrator *migrate.Migrator) error {
	group, err := migrator.Migrate(ctx)
	if err != nil {
		return err
	}
	if group.IsZero() {
		config.Log().Info("there are no new migrations to run (database is up to date)")
		return nil
	}
	config.Log().Infof("migrated to %s", group)

	return nil
}

// Rollback rollbacks the last migration.
func Rollback(ctx context.Context, migrator *migrate.Migrator) error {
	group, err := migrator.Rollback(ctx)
	if err != nil {
		return err
	}
	if group.IsZero() {
		config.Log().Info("there are no groups to roll back")
		return nil
	}
	config.Log().Infof("rollbacked %s", group)
	return nil
}

// Status shows current migration group
func Status(ctx context.Context, migrator *migrate.Migrator) error {
	ms, err := migrator.MigrationsWithStatus(ctx)
	if err != nil {
		return err
	}
	config.Log().Infof("migrations: %s", ms)
	config.Log().Infof("unapplied migrations: %s", ms.Unapplied())
	config.Log().Infof("last migration group: %s", ms.LastGroup())

	return nil
}
