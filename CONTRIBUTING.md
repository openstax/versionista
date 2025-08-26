# CONTRIBUTING.md

This file provides guidance to constributors when working with code in this repository.

## Commands

### Build and Development
- Build the binary: `go build`
- Run tests: `go test ./...`
- Run specific test file: `go test -v ./repository_test.go` or `go test -v ./versionista_test.go`

### Running the Application
- Release all repos in a project: `./versionista release <project-name>`
- Review latest versions: `./versionista review <project-name>`
- Release specific repo: `./versionista release organization/repo-name`

## Architecture Overview

Versionista is a Go CLI tool that automates GitHub releases by analyzing commits and pull requests since the last release. The architecture consists of:

### Core Components
- **main.go**: Entry point, configures CLI using Cobra and loads config via Viper
- **commands.go**: CLI command handlers (`release` and `review`) with concurrent repository processing
- **client.go**: GitHub API client wrapper using OAuth2 authentication
- **repository.go**: Repository operations including version resolution, changelog generation, and PR analysis
- **release.go**: Release creation logic supporting regular releases, pre-releases, and hotfixes
- **prompts.go**: Interactive prompts for version selection and release confirmation
- **util.go**: Shared utilities and helper functions

### Key Data Structures
- **Repository**: Manages GitHub repo connection, version tracking, and changelog generation
- **Release**: Represents a release with version and repository reference
- **ChangeLogEntry**: Contains PR metadata (number, date, author, title, JIRA tickets)

### Configuration
Uses `~/.versionista.yml` with:
- `gh_token`: GitHub personal access token
- `projects`: Map of project names to lists of repository configurations
  - Each repository config contains: `repo` (owner/name), `alias` (display name), `jira` (boolean)
- `jira_boards`: Board names for ticket extraction from PR titles
- `branches`: Custom branch mapping (defaults to "main")

### Release Process Flow
1. Fetch latest release and commits since that release
2. Concurrently process multiple repositories using goroutines
3. Parse PR information from commit messages
4. Extract JIRA tickets from PR titles using configurable board patterns
5. Generate changelog and determine version bump (patch/minor/major)
6. Create GitHub release with generated changelog

### Testing
- `repository_test.go`: Repository operations and version parsing
- `versionista_test.go`: Main application logic and integration tests

The tool supports three release types: regular releases from main branch, pre-release fixes, and post-release fixes, with interactive prompts for version selection and changelog editing.
