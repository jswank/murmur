package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	cli "github.com/urfave/cli/v2"
)

const jsonnetDesc = `Work with Jsonnet files.

A list of .jsonnet files can be supplied on the commandline
`

const jsonnetRenderDesc = `Render Jsonnet files.

The jsonnet application is invoked to render files.  The JSONNET_PATH variable
should be set appropriately.  Commandline arguments can be passed to jsonnet
using the 'jsonnet_args' flag.

`

const jsonnetCreateDesc = `Create a new Jsonnet file.

App specific files located in $DATADIR/tmpl/<app>.jsonnet.tmpl are copied to the team/app/env directory specified.

Variables that can be used in templates include TEAM, ENV, and APP.

`

var JsonnetCommand = &cli.Command{
	Name:            "jsonnet",
	Usage:           "work with jsonnet files",
	UsageText:       "murmur jsonnet [options] render [target_files...]",
	HideHelpCommand: true,
	Args:            true,
	ArgsUsage:       "files...",
	Description:     jsonnetDesc,
	Subcommands: []*cli.Command{
		{
			Name:        "create",
			Usage:       "create a new jsonnet file",
			Action:      createJsonnet,
			Description: jsonnetCreateDesc,
			Args:        true,
			ArgsUsage:   "team/app/env",
			Before:      BeforeFunc,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "datadir",
					Usage: "Recursively search this for files to process. Defaults to $DATADIR",
					Value: os.Getenv("DATADIR"),
				},
				&cli.StringFlag{
					Name:  "loglevel",
					Usage: "Set the log level (debug, info, warn, error, fatal, panic)",
					Value: "error",
				},
			},
		},
		{
			Name:   "list",
			Usage:  "list jsonnet files",
			Action: listJsonnet,
			Flags:  DefaultFlags,
			Before: BeforeFunc,
		},
		{
			Name:   "render",
			Usage:  "render jsonnet files",
			Action: renderJsonnet,
			Before: BeforeFunc,
			Flags: append(DefaultFlags,
				&cli.StringFlag{
					Name:  "destdir",
					Usage: "Destination directory for rendered files. If unset, it defaults the same directory as the jsonnet file. The directory is relative to the current working directory.",
					Value: "",
				},
				// this value is depdend on the value of the destdir flag
				&cli.StringFlag{
					Name:  "jsonnet_args",
					Usage: "Arguments to pass to the jsonnet application. Defaults to '-m <destdir>'",
					Value: "-m",
				},
			),
			Description: jsonnetRenderDesc,
		},
	},
}

func listJsonnet(ctx *cli.Context) error {

	files, err := getFiles(ctx, ctx.String("datadir"), ".jsonnet")
	if err != nil && ctx.Bool("errexit") {
		return err
	}

	for _, file := range files {
		fmt.Println(file)
	}

	return nil

}

// create a jsonnet file from a simple template
func createJsonnet(ctx *cli.Context) error {

	var err error

	// parse team/app/env from args
	elem := strings.SplitN(ctx.Args().First(), "/", 3)
	if len(elem) != 3 {
		return fmt.Errorf("team/app/env must be specified")
	}

	input := map[string]string{
		"TEAM": elem[0],
		"APP":  elem[1],
		"ENV":  elem[2],
	}

	// parse template for app
	log.Debug("parsing template", "file", filepath.Join(ctx.String("datadir"), "tmpl", input["APP"]+"jsonnet.tmpl"))
	tmpl, err := template.ParseFiles(filepath.Join(ctx.String("datadir"), "tmpl", fmt.Sprintf("%s.jsonnet.tmpl", input["APP"])))
	if err != nil {
		return err
	}

	// create the directory structure
	dir := filepath.Join(ctx.String("datadir"), input["TEAM"], input["APP"], input["ENV"])
	if err = os.MkdirAll(dir, 0750); err != nil {
		return err
	}
	log.Info("creating directory", "dir", dir)

	// create the app.jsonnet file
	dst := filepath.Join(dir, input["APP"]+".jsonnet")
	log.Info("creating file", "file", dst)
	f, err := os.Create(dst)
	if err != nil {
		return err
	}

	if err = tmpl.Execute(f, input); err != nil {
		return err
	}

	return nil
}

// renderJsonnet renders files from the specified jsonnet files
func renderJsonnet(ctx *cli.Context) error {

	files, err := getFiles(ctx, ctx.String("datadir"), ".jsonnet")
	if err != nil && ctx.Bool("errexit") {
		return err
	}

	renderDir := ctx.String("destdir")
	if renderDir == "" {
		renderDir = "."
	}
	log.Debug("rendering jsonnet files", "files", files, "destdir", renderDir)

	if ctx.String("jsonnet_args") == "-m" {
		ctx.Set("jsonnet_args", "-m "+renderDir)
	}
	jsonnetArgs := strings.Fields(ctx.String("jsonnet_args"))

	for _, file := range files {
		log.Info("jsonnet", "file", file)
		cmd := exec.Command("jsonnet", append(jsonnetArgs, filepath.Base(file))...)
		cmd.Dir = filepath.Dir(file)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		cmd.Stdout = os.Stdout

		log.Info("jsonnet", "cmd", cmd.String(), "dir", cmd.Dir)

		err = cmd.Run()
		if err != nil {
			if ctx.Bool("errexit") {
				return err
			} else {
				log.Warn("jsonnet", "cmd", cmd.String(), "file", file, "msg", err, "stderr", stderr.String())
			}
		}
	}

	return nil
}
