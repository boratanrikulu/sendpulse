package main

import (
	"context"

	"github.com/boratanrikulu/sendpulse/internal/config"
	"github.com/boratanrikulu/sendpulse/internal/db"
	"github.com/boratanrikulu/sendpulse/internal/db/migrator"
	"github.com/boratanrikulu/sendpulse/internal/db/migrator/migrations"

	"github.com/uptrace/bun/migrate"
	"github.com/urfave/cli/v2"
)

func databaseCMD() *cli.Command {
	return &cli.Command{
		Name:    "database",
		Aliases: []string{"db", "d"},
		Usage:   "database migrations",
		Subcommands: []*cli.Command{
			{
				Name:  "init",
				Usage: "Creates migration tables",
				Action: func(c *cli.Context) error {
					path := c.String("config")
					cfg, err := config.NewConfig(path)
					if err != nil {
						return err
					}

					dbc, err := db.Connect(cfg.Database.DSN)
					if err != nil {
						return err
					}
					cfg.SetDB(dbc)

					return migrator.InitMigrator(
						context.Background(), migrate.NewMigrator(dbc, migrations.Migrations))
				},
			},
			{
				Name:  "migrate",
				Usage: "Migrates db to latest migration",
				Action: func(c *cli.Context) error {
					path := c.String("config")
					cfg, err := config.NewConfig(path)
					if err != nil {
						return err
					}
					dbc, err := db.Connect(cfg.Database.DSN)
					if err != nil {
						return err
					}
					cfg.SetDB(dbc)

					return migrator.Migrate(
						context.Background(), migrate.NewMigrator(dbc, migrations.Migrations))
				},
			},
			{
				Name:  "rollback",
				Usage: "Rollbacks db the latest migration",
				Action: func(c *cli.Context) error {
					path := c.String("config")
					cfg, err := config.NewConfig(path)
					if err != nil {
						return err
					}
					dbc, err := db.Connect(cfg.Database.DSN)
					if err != nil {
						return err
					}
					cfg.SetDB(dbc)

					return migrator.Rollback(
						context.Background(), migrate.NewMigrator(dbc, migrations.Migrations))
				},
			},
			{
				Name:  "status",
				Usage: "Shows current migration status",
				Action: func(c *cli.Context) error {
					path := c.String("config")
					cfg, err := config.NewConfig(path)
					if err != nil {
						return err
					}
					dbc, err := db.Connect(cfg.Database.DSN)
					if err != nil {
						return err
					}
					cfg.SetDB(dbc)

					return migrator.Status(
						context.Background(), migrate.NewMigrator(dbc, migrations.Migrations))
				},
			},
			{
				Name:  "seed",
				Usage: "Generate random message data for testing",
				Action: func(c *cli.Context) error {
					count := c.Int("count")
					path := c.String("config")

					cfg, err := config.NewConfig(path)
					if err != nil {
						return err
					}

					dbc, err := db.Connect(cfg.Database.DSN)
					if err != nil {
						return err
					}
					cfg.SetDB(dbc)

					return seedMessages(context.Background(), dbc, count)
				},
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "count",
						Aliases: []string{"n"},
						Usage:   "Number of random messages to generate",
						Value:   10,
					},
				},
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "config.yaml file location",
				Value:   "./configs/sendpulse.yaml",
			},
		},
	}
}
