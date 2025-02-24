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
		Name:  "output",
		Usage: "Set the (log) output to 'json' or 'text'",
		Value: "text",
	},
	&cli.StringFlag{
		Name:  "datadir",
		Usage: "Search path for files. Defaults to '.', can be set using $DATADIR",
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
		Usage: "Limit processing based on a 'team/app/env' string. Overrides team, app, env flag.",
	},
	&cli.BoolFlag{
		Name:  "errexit",
		Usage: "Exit on errors",
	},
}
