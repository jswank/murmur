package cmd

import (
	"os"

	cli "github.com/urfave/cli/v2"
)

const GenerateDesc = `Render and write configuration files.

Combines the subcommands:

	- jsonnet render
	- repos clone
	- repos write
	- repos commit (if --commit is specified)

This command will always render jsonnet files to a single directory before
cloning, writing, and (optionally) committing the repos. The --destdir flag can
be used to specify this directory: if unset, a temporary directory is created,
used, and deleted.
`

var GenerateCommand = &cli.Command{
	Name:            "generate",
	Usage:           "render and write config files",
	UsageText:       "murmur [options] [target_files...]",
	HideHelpCommand: true,
	Args:            true,
	ArgsUsage:       "files...",
	Action:          GenerateFunc,
	Description:     GenerateDesc,
	Flags: append(DefaultFlags,
		&cli.StringFlag{
			Name:  "repodir",
			Usage: "Location of git repos. Defaults to current working directory, can be set with $REPODIR",
			Value: os.Getenv("REPODIR"),
		},
		&cli.StringFlag{
			Name:  "destdir",
			Usage: "Destination directory for rendered files. Defaults the same directory as the jsonnet file, can be set using $DESTDIR.",
			Value: os.Getenv("DESTDIR"),
		},
		&cli.BoolFlag{
			Name:  "overwrite",
			Usage: "Overwrite existing repos with fresh clones",
		},
		&cli.BoolFlag{
			Name:  "commit",
			Usage: "Commit / push changes to git repos",
		},
		&cli.StringFlag{
			Name:  "commit-script",
			Usage: "Script to run to commit / push changes to the repo",
		},
		&cli.StringFlag{
			Name:  "commit-msg",
			Usage: "Commit message",
			Value: "murmur commit",
		},
		&cli.StringFlag{
			Name:  "jsonnet-args",
			Usage: "Arguments to pass to the jsonnet application.",
			Value: "-m",
		},
		&cli.StringFlag{
			Name:   "delete-destdir",
			Usage:  "Delete the dest dir",
			Hidden: true,
		},
	),
	Before: func(c *cli.Context) error {

		// override to exit on error for this command
		c.Set("errexit", "true")

		// this will be set to true if the destdir is created later:
		c.Set("delete-destdir", "false")

		return BeforeFunc(c)
	},
	After: func(c *cli.Context) error {
		if c.String("delete-destdir") == "true" {
			log.Debug("Deleting destdir", "dir", c.String("destdir"))
			err := os.RemoveAll(c.String("destdir"))
			if err != nil {
				return err
			}
		}
		return nil
	},
}

func GenerateFunc(c *cli.Context) error {

	// create a temporary directory if destdir is not set
	if c.String("destdir") == "" {
		destdir, err := os.MkdirTemp("", "murmur")
		if err != nil {
			return err
		}
		log.Debug("Created temp destdir", "dir", destdir)

		c.Set("destdir", destdir)
		c.Set("delete-destdir", "true")
	}

	err := renderJsonnet(c)
	if err != nil {
		return err
	}

	// renderJsonnet (may have) used datadir to find jsonnet files.  Subsequent
	// commands use the rendered files: override datadir to point to the destdir
	// so that these files are used.

	c.Set("datadir", c.String("destdir"))

	// set filter to * to avoid filtering out any files in the destdir: filters
	// have already been applied in order to render these files
	c.Set("filter", "")

	err = cloneRepos(c)
	if err != nil {
		return err
	}

	err = writeRepos(c)
	if err != nil {
		return err
	}

	if c.Bool("commit") {
		err = commitRepos(c)
		if err != nil {
			return err
		}
	}

	return nil

}
