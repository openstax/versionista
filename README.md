# versionista : A simple CLI app to cut releases on GitHub

## [![Build Status](https://travis-ci.org/openstax/versionista.svg?branch=master)](https://travis-ci.org/openstax/versionista)

![release screenshot](screenshots/release.png?raw=true "Release Screenshot")

Versionista is a modular Go CLI tool that automates GitHub releases by analyzing commits and pull requests since the last release. It supports project-based releases with cross-repository linking and individual repository management.

## Features

- **Interactive Release Management**: Default interactive mode with version bump selection menu (Skip, Patch, Minor, Major)
- **Automated Release Creation**: Analyzes commits since the last release and creates new releases
- **Pull Request Integration**: Extracts PR information and generates structured table-format changelogs  
- **Cross-Repository Linking**: Links related releases across repositories in the same project (excludes self-references)
- **JIRA Integration**: Extracts and includes JIRA ticket references from PR descriptions
- **Smart Repository Handling**: Automatically handles repositories without releases (defaults to v0.0.0, uses the last 10 merged PRs)
- **Table-Format Release Notes**: Generates clean markdown tables with collapsible PR descriptions
- **Flexible Configuration**: Support both project-based and individual repository releases
- **Semantic Versioning**: Interactive version bumping with semantic versioning support
- **Configurable Logging**: Multiple log levels (debug, info, warn, error) for different use cases

## How It Works

When executed, versionista:

1. Finds the latest release for each configured repository
2. Compares the current branch against the last release to detect changes  
3. Analyzes commits to extract pull request information
4. Generates structured release notes using GitHub's collapsed sections format
5. Creates new GitHub releases with automatic version bumping
6. Optionally cross-links related repository releases within the same project

## Install

Install [from a release](https://github.com/openstax/versionista/releases)

or build manually by checking out code and running `go build` in the source directory

## Example

Create a `.versionista.yml` configuration file. The file will be searched in this order:
1. Custom path specified with `-c/--config` flag  
2. `~/.versionista.yml` (home directory)
3. `./.versionista.yml` (current working directory)

Example configuration:

```
gh_token: <git hub personal api token>
projects:
  <project name>:
    - repo: repo-organization/repo-name
      alias: MyCustomName
      jira: true
      crossLink: true
      generate-assets: ./scripts/build-release.sh
      path: /path/to/local/checkout
    - repo: repo-organization/other-repo
      alias: OtherName
      jira: false
      crossLink: false

jira_boards:
  - board-for-project-one
  - board-for-project-two


branches:
  repo-organization/repo-name: feature-branch

```

### Configuration Options

- **alias**: Custom display name for the repository (optional)
- **jira**: Enable/disable JIRA ticket extraction from PR descriptions (default: true)
- **crossLink**: Enable/disable cross-linking to other repositories in the project within release notes (default: false)
- **generate-assets**: Shell command to build release assets (optional)
- **path**: Local checkout directory the `generate-assets` command runs in (optional; defaults to the directory versionista was invoked from)
- **jira_org_id**: Top-level Atlassian organization id used to build JIRA ticket links; required when any repository has `jira` enabled

When `crossLink` is enabled for a repository, its release notes will include a "Related Releases" section at the top with links to all repositories in the same project, showing either the newly released version or the latest existing version if no release was made.

When `generate-assets` is set, the command runs before the release is created (so a failure aborts the release instead of leaving an empty one). It runs with the default shell in the repository's `path` directory, with the chosen version passed as the first argument (`$1`). Before running, versionista verifies the git working tree is clean and checks out the commit being released; once the command finishes it restores the branch or commit that was checked out beforehand. If the command succeeds, each line of its standard output is treated as a file path (relative paths are resolved against `path`) and uploaded to the release as an asset.

## Commands

### Basic Usage

* **release** all repos for a project: `versionista release <project name>`
* **release** a single repo within a project: `versionista release <project name> --repo <repo-name>`
* **review** render an HTML changelog preview for a project and open it in the browser: `versionista review <project name>`
* **hotfix** cut a hotfix release for one repo from a specific commit: `versionista hotfix <repository> <sha>`
* **append** extend an existing release with newer commits and move its tag: `versionista append <repository> <release-tag> <sha>`

Alternatively you can release or review any repository even if it's not listed by using the `organization/name` format like:
`versionista release organization/name`

The `--repo` (`-r`) flag limits a release to one repository in the project, matched by its short name (the part after `organization/`). The rest of the project is still loaded so cross-links resolve correctly.

The `hotfix` and `append` commands take a repository by name; the project is auto-detected from the repository (or set explicitly with `--project`).

### CLI Flags

#### Global Flags
Available for all commands:

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--config` | `-c` | Path to configuration file | `~/.versionista.yml` or `./.versionista.yml` |
| `--log-level` | `-l` | Set logging level (debug, info, warn, error) | `warn` |
| `--project` | `-p` | Specify the project to use | (auto-detected) |
| `--dry-run` | | Perform a dry run without creating actual releases | `false` |
| `--help` | `-h` | Show help information | |

#### Release Command Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--repo` | `-r` | Release only the named repository within the project | (all repos) |

All releases use interactive mode by default.

### Usage Examples

#### Interactive Mode (Default)
```bash
# Interactive release with version selection menu
versionista release myproject

# Release only one repository within a project
versionista release myproject --repo repo-name

# Interactive release for specific repository
versionista release organization/repo-name

# Interactive with debug logging
versionista release myproject --log-level debug
```

#### Hotfix and Append
```bash
# Cut a hotfix release for one repo from a specific commit
versionista hotfix repo-name 1a2b3c4

# Append newer commits to an existing release and move its tag
versionista append repo-name v1.2.3 1a2b3c4
```


#### Review Commands
```bash
# Review latest versions
versionista review myproject

# Review with custom configuration
versionista review myproject --config /path/to/config.yml --log-level info
```

### Release Modes

**Interactive Mode (Default)**: 
- Presents a menu to select version bump type (Skip, Patch, Minor, Major)
- Shows recent pull requests since last release
- Allows manual decision-making for each repository
- Ideal for manual releases and version planning


### Release Notes Format

Versionista generates clean, structured release notes in markdown table format:

```markdown
## Related Releases

- [backend v1.2.3](https://github.com/org/backend/releases/tag/v1.2.3)
- [shared-lib v0.5.0](https://github.com/org/shared-lib/releases/tag/v0.5.0)

---

| PR # | Author | Title | Merged Date | Ticket # |
|------|--------|-------|-------------|----------|
| #123 | johndoe | <details><summary>Add new feature</summary><br>This PR adds a comprehensive new feature that improves user experience...</details> | 2023-12-01 | PROJ-456 |
| #124 | janedoe | Fix critical bug | 2023-12-02 | PROJ-457, PROJ-458 |
```

**Features:**
- **Collapsible Descriptions**: PR descriptions are hidden by default using HTML `<details>` tags
- **Cross-Repository Links**: Shows related releases (excludes the current repository)
- **JIRA Integration**: Automatically extracts and displays ticket numbers
- **Escaped Content**: Handles special markdown characters safely
- **Clean Layout**: Easy-to-read table format with consistent columns

## Architecture

Versionista is a single `main` package with each file owning a distinct concern:

### File Structure

```
├── main.go          # Application entry point
├── commands.go      # Cobra CLI command handlers (release, review, hotfix, append)
├── config.go        # Config loading and validation (.versionista.yml)
├── client.go        # GitHub API client wrapper
├── release.go       # Release processing and orchestration
├── changelog.go     # Changelog generation and table formatting
├── assets.go        # generate-assets command execution and asset upload
├── prompts.go       # Interactive version-bump prompts
├── review_html.go   # HTML changelog preview rendering
├── version.go       # Semantic version parsing and bumping
└── logger.go        # Leveled logging
```

### Core Components

- **config.go**: Loads configuration from `.versionista.yml`, validates it, and provides access to project settings
- **client.go**: Wraps the GitHub API client with domain-specific functionality for repositories, releases, pull requests, and asset uploads
- **release.go**: Orchestrates the release process from version resolution to release creation, cross-linking, and asset generation
- **changelog.go**: Generates release notes from pull request data with JIRA ticket extraction and GitHub's collapsed sections format
- **assets.go**: Runs a repository's `generate-assets` command and uploads the resulting files to the release
- **version.go**: Handles semantic version parsing, validation, and bumping (patch, minor, major)
- **logger.go**: Provides leveled logging (Debug, Info, Warn, Error, Fatal)

### Design Principles

- **Error Handling**: Comprehensive error handling with context-aware error messages
- **Testability**: Core logic is covered by unit tests (`*_test.go`)
- **Configuration-Driven**: Flexible configuration system supporting different project structures
- **Concurrency**: Repository version resolution is processed concurrently for improved performance

## Author

Nathan Stitt

## License

MIT.
