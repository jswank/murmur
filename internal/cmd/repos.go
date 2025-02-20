package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jswank/murmur/pkg/murmur"

	cli "github.com/urfave/cli/v2"
)

const ReposDesc = `Work with repos.

A list of target.json files can be supplied on the commandline
`

var ReposCommand = &cli.Command{
	Name:            "repos",
	Usage:           "work with repos",
	UsageText:       "murmur repos [options] list|write [target_files...]",
	HideHelpCommand: true,
	Args:            true,
	ArgsUsage:       "files...",
	// Action:          ReposFunc,
	Description: ReposDesc,
	Subcommands: []*cli.Command{
		{
			Name:   "list",
			Usage:  "list repos",
			Action: listRepos,
			Flags:  DefaultFlags,
			Before: BeforeFunc,
		},
		{
			Name:   "clone",
			Usage:  "clone repos",
			Action: cloneRepos,
			Before: BeforeFunc,
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
			),
		},
		{
			Name:   "write",
			Usage:  "write to repos",
			Action: writeRepos,
			Before: BeforeFunc,
			Flags: append(DefaultFlags,
				&cli.StringFlag{
					Name:  "repodir",
					Usage: "location of git repos. Defaults to $REPODIR",
					Value: os.Getenv("REPODIR"),
				}),
		},
		{
			Name:   "commit",
			Usage:  "commit repos",
			Action: commitRepos,
			Before: BeforeFunc,
			Flags: append(DefaultFlags, []cli.Flag{
				&cli.StringFlag{
					Name:  "repodir",
					Usage: "location of git repos. Defaults to $REPODIR",
					Value: os.Getenv("REPODIR"),
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
			}...),
		},
	},
}

// listRepos prints a list of unique repos from a list of target files
func listRepos(ctx *cli.Context) error {

	files, err := getFiles(ctx, ctx.String("datadir"), "targets.json")
	if err != nil && ctx.Bool("errexit") {
		return err
	}

	targets, err := getTargets(files)
	if err != nil {
		return err
	}

	repos := make(map[string]bool)
	for _, target := range targets {
		if _, ok := repos[target.Name+target.Branch]; ok {
			continue
		}
		fmt.Printf("%s:%s\n", target.Repo, target.Branch)
		repos[target.Name+target.Branch] = true
	}

	return nil

}

// cloneRepos clones the repos from a list of target files
func cloneRepos(ctx *cli.Context) error {

	files, err := getFiles(ctx, ctx.String("datadir"), "targets.json")
	if err != nil && ctx.Bool("errexit") {
		return err
	}

	targets, err := getTargets(files)
	if err != nil {
		return err
	}

	// create the repodir if it doesn't exist
	err = os.MkdirAll(ctx.String("repodir"), 0755)
	if err != nil {
		return err
	}

	cloned_repos := make(map[string]bool)

	for _, target := range targets {
		// clone the repo if it hasn't been cloned yet
		if _, ok := cloned_repos[target.Name+target.Branch]; ok {
			continue
		}
		cloned_repos[target.Name+target.Branch] = true

		// if the repo would be cloned to an already existing directory, log an error unless '--overwrite' is set
		if _, err = os.Stat(filepath.Join(ctx.String("repodir"), target.CloneDir())); err == nil {
			if !ctx.Bool("overwrite") {
				log.Error("repository already cloned", "repo", target.Name, "branch", target.Branch)
				continue
			} else {
				// remove the existing clone directory
				log.Debug("removing existing repo", "repo", target.Name, "branch", target.Branch, "dir", filepath.Join(ctx.String("repodir"), target.CloneDir()))
				err = os.RemoveAll(filepath.Join(ctx.String("repodir"), target.CloneDir()))
				if err != nil {
					return fmt.Errorf("unable to remove existing repo, %w", err)
				}
			}
		}
		err := cloneTargetRepo(ctx.String("repodir"), target)
		if err != nil {
			log.Error("unable to clone repository", "repo", target.Name, "branch", target.Branch, "error", err)
			if ctx.Bool("errexit") {
				return fmt.Errorf("unable to clone repository %s", target.Name)
			}
		}
	}

	return nil

}

// writeRepos writes generated files to the targeted repositories
func writeRepos(ctx *cli.Context) error {

	files, err := getFiles(ctx, ctx.String("datadir"), "targets.json")
	if err != nil {
		return err
	}

	targets, err := getTargets(files)
	if err != nil {
		return err
	}

	err = writeFilesToRepos(ctx.String("repodir"), targets)
	if err != nil {
		return err
	}

	return nil

}

// commitRepos commits changes to repos and pushes them upstream
func commitRepos(ctx *cli.Context) error {

	files, err := getFiles(ctx, ctx.String("datadir"), "targets.json")
	if err != nil {
		return err
	}

	targets, err := getTargets(files)
	if err != nil {
		return err
	}

	committed_repos := make(map[string]bool)

	// commit the repo if it hasn't been committed yet
	for _, target := range targets {
		if _, ok := committed_repos[target.Name+target.Branch]; ok {
			continue
		}
		err := commitTargetRepo(ctx, target)
		if err != nil {
			return err
		}
		committed_repos[target.Name+target.Branch] = true
	}

	return nil

}

// commit a single repository from a target
func commitTargetRepo(ctx *cli.Context, target murmur.Target) error {

	commitScript := ctx.String("commit_script")
	commitMsg := ctx.String("commit_msg")
	repodir := ctx.String("repodir")

	var err error

	// check if the repository has already been cloned in repodir / target.Name
	if _, err = os.Stat(filepath.Join(repodir, target.CloneDir())); err != nil {
		return fmt.Errorf("repository not cloned, %w", err)
	}

	// if a commit script is provided, run it rather than our default commit process
	if commitScript != "" {
		commitCmd := exec.Command(commitScript)
		commitCmd.Dir = filepath.Join(repodir, target.Name)
		log.Info("running commit script", "repo", target.Repo, "branch", target.Branch, "dir", filepath.Join(repodir, target.Name))
		err = commitCmd.Run()
		if err != nil {
			return fmt.Errorf("unable to run commit script, %w", err)
		}
		return nil
	}

	// commit the repo
	commitCmd := exec.Command("git", "commit", "-am", commitMsg)
	commitCmd.Dir = filepath.Join(repodir, target.Name)
	log.Info("committing repository", "repo", target.Repo, "branch", target.Branch, "dir", filepath.Join(repodir, target.Name))
	err = commitCmd.Run()
	if err != nil {
		return fmt.Errorf("unable to commit repository, %w", err)
	}

	// push repo to the remote origin
	pushCmd := exec.Command("git", "push")
	pushCmd.Dir = filepath.Join(repodir, target.Name)
	log.Info("pushing repository", "repo", target.Repo, "branch", target.Branch, "dir", filepath.Join(repodir, target.Name))
	err = pushCmd.Run()
	if err != nil {
		return fmt.Errorf("unable to push repository, %w", err)
	}

	return nil

}

// clone a single repository from a target
func cloneTargetRepo(repodir string, target murmur.Target) error {

	// check if the repository has already been cloned in repodir / target.Name
	if _, err := os.Stat(filepath.Join(repodir, target.Name)); err == nil {
		log.Info("repository already cloned", "repo", target.Name)
		return nil
	}

	githubURL := fmt.Sprintf("https://%s@github.com/%s.git", os.Getenv("GITHUB_TOKEN"), target.Repo)

	cloneCmd := exec.Command("git", "clone", "--depth", "1", "--branch", target.Branch, githubURL, target.CloneDir())
	cloneCmd.Dir = repodir

	log.Info("cloning repository", "repo", target.Repo, "branch", target.Branch, "dir", filepath.Join(repodir, target.CloneDir()))
	err := cloneCmd.Run()
	if err != nil {
		return fmt.Errorf("unable to clone repository %s, %w", target.Repo, err)
	}

	return nil
}

// writeFilesToRepos writes files to the target repositories
func writeFilesToRepos(repo_dir string, targets []murmur.Target) error {
	for _, target := range targets {
		src_dir := filepath.Dir(target.Filename)
		dest_dir := filepath.Join(repo_dir, target.CloneDir(), target.Path)
		for _, t := range target.Types {
			// return a list of files in the same directory of matching types
			// files are named *-<app>-<type>.json
			files, err := filepath.Glob(filepath.Join(src_dir, fmt.Sprintf("*-%s-%s.json", target.App, t)))
			if err != nil {
				return fmt.Errorf("unable to read files, %w", err)
			}
			log.Info("writing files to repository", "src", src_dir, "dest", dest_dir, "type", t, "files", files)
			for _, file := range files {
				// dest filename is the same as the source filename, minus the <app>. For instance,
				// for the app "pyrenees", filename == "ets-cloudops-infrastructure-pyrenees-datasources.json" and
				// dest_filename == "ets-cloudops-infrastructure-datasources.json"
				filename := filepath.Base(file)
				// remove -app- from the dest_filename
				dest_filename := strings.Replace(filename, fmt.Sprintf("-%s-", target.App), "-", 1)
				// fmt.Printf("cp %s %s\n", file, filepath.Join(dest_dir, t, dest_filename))
				log.Debug("copying file", "file", file, "dest", filepath.Join(dest_dir, t, dest_filename))
				err = copyFile(file, filepath.Join(dest_dir, t, dest_filename))
				if err != nil {
					log.Error("unable to copy file", "file", file, "dest", filepath.Join(dest_dir, t, dest_filename), "error", err)
					return err
				}
			}
		}
	}
	return nil
}
