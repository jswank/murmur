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

A list of target.json files can be supplied on the commandline.

If a target defines repo=".", then the destination will be relative to the
currrent working directory- not a remote git repository.

`

// Shared flags for multiple commands
// branchOverridesFlag is a flag shared by commands that manipulate repositories
var branchOverridesFlag = &cli.StringSliceFlag{
	Name:  "override-branch",
	Usage: "Override branch for specific repo (format: repo_name:branch)",
}

// repoDirFlag is a flag shared by commands that need repository location
var repoDirFlag = &cli.StringFlag{
	Name:  "repodir",
	Usage: "Location of git repos. Defaults to current working directory, can be set with $REPODIR",
	Value: os.Getenv("REPODIR"),
}

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
			Flags:  append(DefaultFlags, branchOverridesFlag),
			Before: BeforeFunc,
		},
		{
			Name:   "clone",
			Usage:  "clone repos",
			Action: cloneRepos,
			Before: BeforeFunc,
			Flags: append(DefaultFlags,
				repoDirFlag,
				branchOverridesFlag,
				&cli.BoolFlag{
					Name:  "overwrite",
					Usage: "Overwrite existing repos with fresh clones",
				},
			),
		},
		{
			Name:   "write",
			Usage:  "write to repos",
			Action: writeRepos,
			Before: BeforeFunc,
			Flags: append(DefaultFlags,
				branchOverridesFlag,
				repoDirFlag,
			),
		},
		{
			Name:   "commit",
			Usage:  "commit repos",
			Action: commitRepos,
			Before: BeforeFunc,
			Flags: append(DefaultFlags,
				branchOverridesFlag,
				repoDirFlag,
				&cli.StringFlag{
					Name:  "commit-script",
					Usage: "script to run to commit the repo",
				},
				&cli.StringFlag{
					Name:  "commit-msg",
					Usage: "commit message",
					Value: "murmur commit",
				},
			),
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

	// Apply branch overrides to all targets at once
	applyBranchOverrides(ctx, targets)

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

	// Apply branch overrides to all targets at once
	applyBranchOverrides(ctx, targets)

	// create the top-level repodir if it doesn't exist
	if ctx.String("repodir") != "" {
		err = os.MkdirAll(ctx.String("repodir"), 0755)
		if err != nil {
			return err
		}
	}

	cloned_repos := make(map[string]bool)

	for _, target := range targets {
		if target.Repo == "." {
			continue
		}

		// clone the repo only if it hasn't been cloned in this loop
		if _, ok := cloned_repos[target.Name+target.Branch]; ok {
			continue
		}
		cloned_repos[target.Name+target.Branch] = true

		err = processExistingCloneDir(ctx, target)
		if err != nil {
			log.Error("unable to process clone directory", "error", err)
		}

		// clone the repository
		err = cloneTargetRepo(ctx.String("repodir"), target)
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

	// Apply branch overrides to all targets at once
	applyBranchOverrides(ctx, targets)

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

	// Apply branch overrides to all targets at once
	applyBranchOverrides(ctx, targets)

	committed_repos := make(map[string]bool)

	// commit the repo if it hasn't been committed yet
	for _, target := range targets {
		if target.Repo == "." {
			log.Info("skipping commit for target with Repo == '.'", "repo", target.Repo, "branch", target.Branch)
			continue
		}
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

	var err error

	commitMsg := ctx.String("commit-msg")
	repoDir := ctx.String("repodir")

	// set commitScript to the absolute path of the script, relative to the
	// current working directory, if it is set
	commitScript := ""
	if ctx.String("commit-script") != "" {
		commitScript, err = filepath.Abs(ctx.String("commit-script"))
		if err != nil {
			return fmt.Errorf("unable to get absolute path of commit script, %w", err)
		}
	}

	cloneDir := filepath.Join(repoDir, target.CloneDir())

	// check if the repository has already been cloned in repodir / target.Name
	if _, err = os.Stat(cloneDir); err != nil {
		return fmt.Errorf("repository not cloned, %w", err)
	}

	// run `git add .` in the repository
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = cloneDir
	log.Info("adding files to repo", "cmd", addCmd.String(), "repo", target.Repo, "branch", target.Branch, "dir", addCmd.Dir)
	err = addCmd.Run()
	if err != nil {
		return fmt.Errorf("unable to add files to repo, %w", err)
	}

	// if git diff --cached --quiet returns 0, there are no changes to commit- exit
	diffCmd := exec.Command("git", "diff", "--cached", "--quiet")
	diffCmd.Dir = cloneDir
	err = diffCmd.Run()
	if err == nil {
		log.Info("no changes to commit to repo", "cmd", diffCmd.String(), "repo", target.Repo, "branch", target.Branch, "dir", diffCmd.Dir)
		return nil
	}

	// commit files to the repository
	commitCmd := exec.Command("git", "commit", "-am", commitMsg)

	// if a commit script is provided, run it rather than our default commit & push process
	if commitScript != "" {
		log.Debug("running commit script", "script", commitScript)
		commitCmd = exec.Command(commitScript)
	}
	commitCmd.Dir = cloneDir
	commitCmd.Stdout = os.Stdout
	commitCmd.Stderr = os.Stderr

	log.Info("commiting changes to repo", "cmd", commitCmd.String(), "repo", target.Repo, "branch", target.Branch, "dir", commitCmd.Dir)
	err = commitCmd.Run()
	if err != nil {
		return fmt.Errorf("Unable to commit to repo. %w", err)
	}

	if commitScript != "" {
		return nil
	}

	// push repo to the remote origin
	pushCmd := exec.Command("git", "push")
	pushCmd.Dir = cloneDir
	log.Info("pushing repository", "repo", target.Repo, "branch", target.Branch, "dir", pushCmd.Dir)
	err = pushCmd.Run()
	if err != nil {
		return fmt.Errorf("unable to push repository, %w", err)
	}

	return nil

}

// clone a single repository from a target
func cloneTargetRepo(repodir string, target murmur.Target) error {

	githubURL := fmt.Sprintf("https://github.com/%s.git", target.Repo)

	if os.Getenv("GITHUB_TOKEN") != "" {
		githubURL = fmt.Sprintf("https://oauth2:%s@github.com/%s.git", os.Getenv("GITHUB_TOKEN"), target.Repo)
	} else {
		log.Warn("$GITHUB_TOKEN is not set: pushes to remote repos will fail unless using an external commit-script")
	}

	log.Debug("githubURL (redacted token)", "url", fmt.Sprintf("https://oauth2:${GITHUB_TOKEN}@github.com/%s.git", target.Repo))

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
		log.Debug("processing target", "repo", target.Repo, "branch", target.Branch, "CloneDir", target.CloneDir())

		target_repo_dir := repo_dir
		if target.Repo == "." {
			log.Debug("overriding repo_dir with current working directory for Repo == '.'", "repo_dir", repo_dir)
			target_repo_dir = "."
		}

		dest_dir := filepath.Join(target_repo_dir, target.CloneDir(), target.Path)
		log.Debug("dest_dir for this target is set", "dest_dir", dest_dir)

		// The toplevel directory (data directory) should already exist.  Return an error if it does not.
		if _, err := os.Stat(dest_dir); err != nil {
			log.Error("destination directory does not exist", "dest_dir", dest_dir, "error", err)
			return err
		} else {
			log.Info("destination directory exists", "dest_dir", dest_dir)
		}

		for _, t := range target.Types {
			// return a list of files in the same directory of matching types
			// files are named *-<app>-<type>.json
			//
			// BUG: if there are multiple targets and app-type-specific files in the same
			// directory, all the matching files will be copied to the target directory
			log.Debug("processing target type", "type", t)
			log.Debug("searching for file glob", "glob", filepath.Join(target.Dir, fmt.Sprintf("%s-%s.json", target.Prefix, t)))
			files, err := filepath.Glob(filepath.Join(target.Dir, fmt.Sprintf("%s-%s.json", target.Prefix, t)))
			if err != nil {
				return fmt.Errorf("unable to read files, %w", err)
			}
			log.Debug("matching files for target type found", "files", files)

			type_dest_dir := filepath.Join(dest_dir, t)
			err = os.MkdirAll(type_dest_dir, 0755)
			if err != nil {
				return fmt.Errorf("unable to create directory, %w", err)
			}

			log.Info("writing files to repository", "src", target.Dir, "dest", type_dest_dir, "type", t)

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

// applyBranchOverrides processes branch override flags (in format repo_name:branch)
// and applies them to all targets where the repo name matches
func applyBranchOverrides(ctx *cli.Context, targets []murmur.Target) {
	// Get branch overrides from the command line
	branchOverrides := ctx.StringSlice("override-branch")
	if len(branchOverrides) == 0 {
		return
	}

	// Process each override
	for _, override := range branchOverrides {
		parts := strings.SplitN(override, ":", 2)
		if len(parts) != 2 {
			log.Warn("invalid branch override format", "override", override, "format", "repo_name:branch")
			continue
		}

		repoName := parts[0]
		branch := parts[1]

		log.Debug("branch override", "repoName", repoName, "branch", branch)

		// Apply override to all matching targets
		for i := range targets {
			log.Debug("checking target for branch override", "target", targets[i].Name, "branch", targets[i].Branch)
			if targets[i].Name == repoName {
				log.Info("overriding branch for repo", "repo", targets[i].Name, "original_branch", targets[i].Branch, "new_branch", branch)
				targets[i].Branch = branch
			}
		}
	}
}

// processExistingCloneDir deals with an existing clone directory for a target
// if '--overwrite' is set, remove the existing directory
func processExistingCloneDir(ctx *cli.Context, target murmur.Target) error {

	_, err := os.Stat(filepath.Join(ctx.String("repodir"), target.CloneDir()))
	if err != nil {
		if os.IsNotExist(err) {
			// directory does not exist, return no error
			return nil
		}
		log.Error("unable to stat repository directory", "repo", target.Name, "branch", target.Branch, "dir", filepath.Join(ctx.String("repodir"), target.CloneDir()), "error", err)
		return err
	}

	log.Warn("repository directory already exists", "repo", target.Name, "branch", target.Branch, "dir", filepath.Join(ctx.String("repodir"), target.CloneDir()))
	if !ctx.Bool("overwrite") {
		err = fmt.Errorf("repository will not be overwritten: specify --overwrite to overwrite existing repos")
	} else {
		// remove the existing clone directory
		log.Debug("removing existing repository directory", "repo", target.Name, "branch", target.Branch, "dir", filepath.Join(ctx.String("repodir"), target.CloneDir()))
		err = os.RemoveAll(filepath.Join(ctx.String("repodir"), target.CloneDir()))
		if err != nil {
			err = fmt.Errorf("unable to remove existing repo, %w", err)
		}
	}

	return err

}
