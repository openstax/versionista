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
- **Smart Repository Handling**: Automatically handles repositories without releases (defaults to v0.0.0, uses last month's PRs)
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

When `crossLink` is enabled for a repository, its release notes will include a "Related Releases" section at the top with links to all repositories in the same project, showing either the newly released version or the latest existing version if no release was made.

## Commands

### Basic Usage

* **release** all repos for a project: `versionista release <project name>`
* **review** display latest versions of all repos in project: `versionista review <project name>`

Alternatively you can release or review any repository even if it's not listed by using the `organization/name` format like:
`versionista release organization/name`

### CLI Flags

#### Global Flags
Available for all commands:

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--config` | `-c` | Path to configuration file | `~/.versionista.yml` or `./.versionista.yml` |
| `--log-level` | `-l` | Set logging level (debug, info, warn, error) | `warn` |
| `--help` | `-h` | Show help information | |

#### Release Command Flags

Currently, all releases use interactive mode by default. No additional flags are available for the release command.

### Usage Examples

#### Interactive Mode (Default)
```bash
# Interactive release with version selection menu
versionista release myproject

# Interactive release for specific repository
versionista release organization/repo-name

# Interactive with debug logging
versionista release myproject --log-level debug
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

Versionista follows a modular architecture with clear separation of concerns:

### Package Structure

```
├── main.go                    # Application entry point
├── commands.go               # CLI command handlers
├── pkg/
│   ├── config/              # Configuration management
│   │   ├── config.go        # Config loading and validation
│   │   └── config_test.go   # Configuration tests
│   ├── github/              # GitHub API integration
│   │   └── client.go        # GitHub client wrapper
│   ├── logging/             # Structured logging
│   │   └── logger.go        # Logger implementation
│   ├── version/             # Semantic versioning
│   │   ├── version.go       # Version parsing and bumping
│   │   └── version_test.go  # Version tests
│   ├── changelog/           # Changelog generation
│   │   ├── changelog.go     # Changelog creation and formatting
│   │   └── changelog_test.go # Changelog tests
│   └── release/             # Release management
│       └── release.go       # Release processing logic
```

### Core Components

- **Config Package**: Handles configuration loading from `.versionista.yml`, validation, and provides access to project settings
- **GitHub Package**: Wraps the GitHub API client with domain-specific functionality for repositories, releases, and pull requests
- **Logging Package**: Provides structured logging with different levels (Info, Error, Debug, Fatal)
- **Version Package**: Handles semantic version parsing, validation, and bumping (patch, minor, major)
- **Changelog Package**: Generates release notes from pull request data with JIRA ticket extraction and GitHub's collapsed sections format
- **Release Package**: Orchestrates the entire release process from version resolution to release creation

### Design Principles

- **Modular Design**: Each package has a single responsibility and well-defined interfaces
- **Error Handling**: Comprehensive error handling with context-aware error messages
- **Testability**: All packages include unit tests with high coverage
- **Configuration-Driven**: Flexible configuration system supporting different project structures
- **Concurrency**: Repository operations are processed concurrently for improved performance

## Author

Nathan Stitt

## License

MIT.
