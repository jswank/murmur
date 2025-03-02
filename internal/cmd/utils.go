package cmd

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jswank/murmur/pkg/murmur"

	cli "github.com/urfave/cli/v2"
)

func readListFromStdin() ([]string, error) {
	var list []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		list = append(list, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// getFiles returns a list of matching files from the commandline arguments
// if 'dir' is specified, it will be searched for matching files
// otherwise, use the first argument as the only file, or read a list from stdin
func getFiles(ctx *cli.Context, dir, suffix string) ([]string, error) {

	var files []string
	var err error

	// if there is an argument, use it to get the list of files
	if ctx.Args().First() != "" {
		// if it is a dash, read the list from stdin
		if ctx.Args().First() == "-" {
			log.Debug("reading list of files from stdin")
			files, err = readListFromStdin()
			if err != nil {
				return nil, fmt.Errorf("unable to read list from stdin, %w", err)
			}
		} else {
			log.Debug("reading list of files from the command line")
			files = ctx.Args().Slice()
		}
	} else if dir != "" {
		log.Debug("searching for files", "dir", dir, "suffix", suffix)
		files, err = findFiles(dir, suffix)
		if err != nil {
			return files, err
		}
		log.Debug("found files", "files", files)
	}

	if ctx.String("filter") != "" {
		filter := filepath.Join(dir, ctx.String("filter"))
		log.Debug("filtering files", "filter", filter)
		files, err = filterFiles(files, filter)
		if err != nil {
			return files, err
		}
	}

	if len(files) == 0 {
		log.Warn("no matching files", "dir", dir, "suffix", suffix, "filter", ctx.String("filter"))
		return files, fmt.Errorf("no files matched the filter")
	}

	log.Debug("getFiles response", "files", files)

	return files, nil
}

// search recursively for suffixed files in the specified directory
func findFiles(dir, suffix string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), suffix) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("unable to search for files: %w", err)
	}
	return files, nil
}

// filterFiles returns a list of files that match the filter
func filterFiles(files []string, filter string) ([]string, error) {
	filter += "/*"
	var filtered []string
	for _, file := range files {
		matched, err := filepath.Match(filter, file)
		if err != nil {
			return filtered, err
		}
		if matched {
			filtered = append(filtered, file)
		}
	}
	return filtered, nil
}

// getTargets reads target files and returns the list of Target structs that
// they contain
func getTargets(files []string) ([]murmur.Target, error) {
	var targets []murmur.Target
	for _, file := range files {
		t, err := murmur.NewTargetsFromFile(file)
		if err != nil {
			log.Error("unable to read target file", "file", file, "error", err)
			continue
		}
		targets = append(targets, t...)
	}
	return targets, nil
}

// Copy a file from src to dst using io.Copy
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	return nil
}

// this function is called before each command, after context is ready
func BeforeFunc(ctx *cli.Context) error {
	var err error

	// configure package logger
	log, err = createLogger(ctx.String("loglevel"), ctx.String("output"))
	if err != nil {
		return fmt.Errorf("unable to create a logger, %w", err)
	}

	// set datadir to "."
	if ctx.String("datadir") == "" {
		ctx.Set("datadir", ".")
	}

	if ctx.String("team") != "*" || ctx.String("app") != "*" || ctx.String("env") != "*" {
		if ctx.String("filter") != "" {
			log.Warn("filter is specified, ignoring team/app/env flags", "filter", ctx.String("filter"))
		} else {
			ctx.Set("filter", fmt.Sprintf("%s/%s/%s", ctx.String("team"), ctx.String("app"), ctx.String("env")))
		}
	}
	log.Info("filter", "filter", ctx.String("filter"))

	// if destdir is a relative path, make it absolute based on the current
	// working directory. An absolute path is required because the directory is
	// passed directly to `jsonnet -m` by default.
	if ctx.String("destdir") != "" && !filepath.IsAbs(ctx.String("destdir")) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("unable to get the current working directory, %w", err)
		}
		ctx.Set("destdir", filepath.Join(cwd, ctx.String("destdir")))
	}

	return nil
}
