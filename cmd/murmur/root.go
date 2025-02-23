package main

import (
	"fmt"
	"log/slog"
	"os"

	cli "github.com/urfave/cli/v2"

	"github.com/jswank/murmur/internal/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))

	app := &cli.App{
		Usage: "Murmur configuration management commands",
		Commands: []*cli.Command{
			cmd.GenerateCommand,
			cmd.ReposCommand,
			cmd.JsonnetCommand,
		},
		// parse --version flag
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "print the version",
			},
		},
		Action: func(c *cli.Context) error {
			if c.Bool("version") {
				fmt.Println("Version: ", version)
				fmt.Println("Build Time: ", date)
				fmt.Println("Git Commit Hash: ", commit)
				return nil
			}
			// print usage
			cli.ShowAppHelp(c)
			return nil
		},
		EnableBashCompletion: true,
	}

	if err := app.Run(os.Args); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

}
