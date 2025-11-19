package main

import (
	"github.com/boratanrikulu/sendpulse/internal/config"
	"github.com/boratanrikulu/sendpulse/internal/db"
	"github.com/boratanrikulu/sendpulse/internal/rest"
	"github.com/boratanrikulu/sendpulse/internal/service"

	"github.com/urfave/cli/v2"
)

func serverCMD() *cli.Command {
	return &cli.Command{
		Name:    "server",
		Aliases: []string{"serve", "s"},
		Usage:   "Starts SendPulse REST API",
		Action: func(c *cli.Context) error {
			path := c.String("config")

			cfg, err := config.NewConfig(path)
			if err != nil {
				return err
			}

			// Connect to database
			dbc, err := db.Connect(cfg.Database.DSN)
			if err != nil {
				return err
			}
			cfg.SetDB(dbc)

			// Initialize services
			messageService := service.NewMessageService(dbc)
			scheduler := service.NewScheduler(dbc, cfg)

			// Auto-start messaging if enabled
			if cfg.Messaging.Enabled {
				if _, err := scheduler.Start(c.Context); err != nil {
					return err
				}
			}

			// Create and start server
			server := rest.NewServer(cfg, messageService, scheduler)
			return server.Start(c.Context)
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
