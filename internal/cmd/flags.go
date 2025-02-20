package cmd

// shared flags for all subcommands
import (
	"os"

	cli "github.com/urfave/cli/v2"
)

var DefaultFlags = []cli.Flag{
	&cli.StringFlag{
		Name:  "loglevel",
		Usage: "Set the log level (debug, info, warn, error, fatal, panic)",
		Value: "error",
	},
	&cli.StringFlag{
		Name:  "datadir",
		Usage: "Recursively search this for files to process. Defaults to $DATADIR",
		Value: os.Getenv("DATADIR"),
	},
	&cli.StringFlag{
		Name:  "team",
		Usage: "Limit processing to team",
		Value: "*",
	},
	&cli.StringFlag{
		Name:  "app",
		Usage: "Limit processing to app",
		Value: "*",
	},
	&cli.StringFlag{
		Name:  "env",
		Usage: "Limit processing to env",
		Value: "*",
	},
	&cli.StringFlag{
		Name:  "filter",
		Usage: "Limit processing to team/app/env. Overrides team, app, env flags",
	},
	&cli.BoolFlag{
		Name:  "errexit",
		Usage: "Exit on errors",
	},
}
