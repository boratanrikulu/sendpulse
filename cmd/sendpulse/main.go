package main

import (
	"log"
	"os"

	_ "github.com/boratanrikulu/sendpulse/docs" // Swagger docs

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "sendpulse",
		Usage: "Robust messaging automation system",
		Commands: []*cli.Command{
			serverCMD(),
			databaseCMD(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
