package cmd

import (
	"os"

	cli "github.com/urfave/cli/v2"
)

const DefaultDesc = `Render and write configuration files.

Combines the subcommands:

	- jsonnet render
	- repos clone
	- repos write
	- repos commit (if --commit is specified)
`

var DefaultCommand = &cli.Command{
	Name:            "generate",
	Usage:           "render and write config files",
	UsageText:       "murmur [options] [target_files...]",
	HideHelpCommand: true,
	Args:            true,
	ArgsUsage:       "files...",
	Action:          DefaultFunc,
	Description:     DefaultDesc,
	// combine DefaultFlags with a Bool flag for --commit
	Flags: append(DefaultFlags,
		&cli.StringFlag{
			Name:  "repodir",
			Usage: "location of git repos. Defaults to $REPODIR",
			Value: os.Getenv("REPODIR"),
		},
		&cli.BoolFlag{
			Name:  "overwrite",
			Usage: "overwrite existing repos",
		},
		&cli.BoolFlag{
			Name:  "commit",
			Usage: "commit changes to git repos",
		},
		&cli.StringFlag{
			Name:  "commit_script",
			Usage: "script to run to commit the repo",
		},
		&cli.StringFlag{
			Name:  "commit_msg",
			Usage: "commit message",
			Value: "murmur commit",
		},
		&cli.StringFlag{
			Name:  "jsonnet_args",
			Usage: "Arguments to pass to the jsonnet application.",
			Value: "-m .",
		},
	),
	Before: BeforeFunc,
}

func DefaultFunc(c *cli.Context) error {
	err := renderJsonnet(c)
	if err != nil {
		return err
	}

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
