# murmur

Orchestrate the generation and deployment of configuration files.

## Quickstart

```bash
# Generate and deploy configuration
murmur generate [options] [target_files...]

# Work with jsonnet files
murmur jsonnet render [options] [jsonnet_files...]

# List, clone, write to, or commit repositories
murmur repos list|clone|write|commit [options] [target_files...]
```

## Usage

Murmur provides a set of commands to manage configuration files using Jsonnet templates and repository management.

### Global Flags

These flags are available for all commands:

- `--loglevel`: Set log level (debug, info, warn, error, fatal, panic) [default: error]
- `--output`: Set log output format (json or text) [default: text]
- `--datadir`: Search path for files [default: current directory or $DATADIR]
- `--team`: Limit processing to specific team [default: *]
- `--app`: Limit processing to specific app [default: *]
- `--env`: Limit processing to specific env [default: *]
- `--filter`: Limit processing based on 'team/app/env' string (overrides team, app, env flags)
- `--errexit`: Exit on errors
- `--version, -v`: Print the version

### Commands

#### generate

Renders and writes configuration files. This is the main command that combines:
- jsonnet render
- repos clone
- repos write
- repos commit (if --commit is specified)

```bash
murmur generate [options] [target_files...]
```

**Flags:**
- `--repodir`: Location of git repos [default: current directory or $REPODIR]
- `--destdir`: Destination directory for rendered files [default: same as jsonnet file or $DESTDIR]
- `--overwrite`: Overwrite existing repos with fresh clones
- `--override-branch value [ --override-branch value ]`:  Override branch for specific repo (format: repo_name:branch)
- `--commit`: Commit and push changes to git repos
- `--commit-script`: Script to run for committing/pushing changes
- `--commit-msg`: Commit message [default: "murmur commit"]
- `--jsonnet-args`: Arguments to pass to jsonnet [default: "-m"]

#### repos

Work with repositories defined in target files.

```bash
murmur repos [subcommand] [options] [target_files...]
```

**Subcommands:**
- `list`: List repositories
- `clone`: Clone repositories
  - Flags: `--repodir`, `--overwrite`
- `write`: Write to repositories
  - Flags: `--repodir`
- `commit`: Commit repositories
  - Flags: `--repodir`, `--commit-script`, `--commit-msg`

#### jsonnet

Work with Jsonnet files.

```bash
murmur jsonnet [subcommand] [options] [jsonnet_files...]
```

**Subcommands:**
- `create`: Create a new jsonnet file
  - Args: "team/app/env"
- `list`: List jsonnet files
- `render`: Render jsonnet files
  - Flags: `--destdir`, `--jsonnet-args`

## Targets

Target files define where configuration should be deployed. Each target specifies:

- `name`: Repository name
- `repo`: Full repository name (e.g., "organization/repo")
- `path`: Top-level destination path for outputs
- `branch`: Git branch name
- `types`: Types of outputs (e.g., "datasources", "connections")
- `app`: Application name

If `target.Repo == .`, then it is assumed that files should be written to the
current directory rather than a repo clone.

### Example Target File

Below is an example target file from the examples directory:

```json
[
   {
      "app": "spacelift",
      "branch": "main",
      "name": "murmur-test",
      "path": "spacelift/data",
      "repo": "jswank/murmur-test",
      "types": [
         "stacks",
         "integrations"
      ]
   }
]
```

This example defines a single target that:
- Works with the `spacelift` application
- Uses the `main` branch of the repository
- Repository name is `murmur-test`
- Writes files to the `spacelift/data` path within the repository
- Full repository name is `jswank/murmur-test`
- Processes two types of outputs: `stacks` and `integrations`
