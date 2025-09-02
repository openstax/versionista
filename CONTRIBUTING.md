# CONTRIBUTING.md

This document provides guidance for contributors working with the versionista codebase.

## Quick Start

### Prerequisites
- Go 1.19 or higher
- Git
- A GitHub personal access token for API access

### Development Setup
1. Clone the repository: `git clone https://github.com/openstax/versionista.git`
2. Navigate to project: `cd versionista`
3. Install dependencies: `go mod tidy`
4. Create a test configuration file (see Configuration section)
5. Build the application: `go build`

### Build and Development Commands
- **Build the binary**: `go build`
- **Run all tests**: `go test ./pkg/...`
- **Run tests with coverage**: `go test -cover ./pkg/...`
- **Run specific package tests**: `go test -v ./pkg/config`
- **Format code**: `go fmt ./...`
- **Lint code** (if golint installed): `golint ./...`

### Running the Application

#### Basic Commands
- **Release all repos in a project**: `./versionista release <project-name>`
- **Review latest versions**: `./versionista review <project-name>`
- **Release specific repo**: `./versionista release organization/repo-name>`
- **Get help**: `./versionista --help`

#### CLI Flags and Options

##### Global Flags (Available for all commands)

| Flag | Short | Description | Default | Example |
|------|-------|-------------|---------|---------|
| `--config` | `-c` | Path to configuration file | `~/.versionista.yml` or `./.versionista.yml` | `--config /path/to/config.yml` |
| `--log-level` | `-l` | Set logging level | `warn` | `--log-level debug` |
| `--help` | `-h` | Show help information | | `--help` |

**Log Levels:**
- `debug`: Verbose output for troubleshooting
- `info`: General information messages
- `warn`: Warning messages (default)
- `error`: Only error messages

##### Release Command Flags

Currently, all releases use interactive mode by default. No additional flags are available for the release command.

#### Development Examples

##### Interactive Development (Recommended for manual testing)
```bash
# Interactive release with debug logging
./versionista release myproject --log-level debug

# Interactive release for specific repository
./versionista release organization/repo-name --log-level info

# Test with custom configuration
./versionista release myproject --config ./test-config.yml
```


##### Debugging and Troubleshooting
```bash
# Maximum verbosity for debugging
./versionista release myproject --log-level debug

# Review mode with detailed logging
./versionista review myproject --log-level debug

# Test configuration loading
./versionista --help --log-level debug
```

#### Development Workflow with Flags

1. **During Development**: Use interactive mode with debug logging
   ```bash
   ./versionista release test-project --log-level debug
   ```

2. **Testing Changes**: Use interactive mode with different log levels
   ```bash
   ./versionista release test-project --log-level info
   ```

3. **Configuration Testing**: Test different config files
   ```bash
   ./versionista release test-project --config ./configs/test.yml
   ```

## Architecture Overview

Versionista follows a modular architecture with clear separation of concerns. The application is organized into focused packages that handle specific responsibilities.

### Modular Package Structure

```
├── main.go                    # Application entry point and configuration loading
├── commands.go               # CLI command handlers and application orchestration
├── backup/                   # Legacy code (ignore for development)
├── pkg/                      # Core application packages
│   ├── config/              # Configuration management
│   ├── github/              # GitHub API integration  
│   ├── logging/             # Structured logging
│   ├── version/             # Semantic versioning
│   ├── changelog/           # Changelog generation
│   └── release/             # Release management
```

### Package Responsibilities

- **config**: Configuration loading from `.versionista.yml`, validation, and access to project settings
- **github**: GitHub API client wrapper with repository, release, and pull request operations  
- **logging**: Structured logging with multiple levels (Info, Error, Debug, Fatal)
- **version**: Semantic version parsing, validation, and version bumping logic
- **changelog**: Release notes generation from PR data with JIRA integration and GitHub formatting
- **release**: Complete release process orchestration from version resolution to release creation

### Configuration

Configuration file locations (searched in this order):
1. Custom path specified with `-c/--config` flag
2. `~/.versionista.yml` (home directory)
3. `./.versionista.yml` (current working directory)

Configuration format:
```yaml
gh_token: <github-personal-access-token>
projects:
  project-name:
    - repo: owner/repo-name
      alias: Display Name
      jira: true
      crossLink: true
jira_boards:
  - BOARD1
  - BOARD2
branches:
  owner/repo: custom-branch
```

### Design Principles

1. **Single Responsibility**: Each package handles one specific concern
2. **Dependency Injection**: Dependencies are passed explicitly, making testing easier
3. **Error Handling**: Comprehensive error handling with context
4. **Interface-Based Design**: Packages depend on interfaces, not concrete implementations
5. **Testability**: All packages include comprehensive unit tests
6. **Concurrent Processing**: Repository operations are processed concurrently for performance

### Development Workflow

1. Make changes to the appropriate package
2. Add or update tests for your changes
3. Run tests: `go test ./pkg/...`
4. Format code: `go fmt ./...`
5. Build and test the application: `go build && ./versionista --help`

## Testing

### Test Organization
- Each package includes comprehensive unit tests in `*_test.go` files
- Tests focus on package-specific functionality and use table-driven test patterns
- Mock implementations are used for external dependencies (GitHub API, file system)

### Running Tests
```bash
# Run all package tests
go test ./pkg/...

# Run tests with coverage report
go test -cover ./pkg/...

# Run tests for a specific package
go test -v ./pkg/config

# Run tests with race detection
go test -race ./pkg/...
```

### Test Coverage Goals
- Aim for >80% test coverage for all packages
- Focus on testing business logic and error conditions
- Include edge cases and boundary conditions in tests

### Writing Tests
- Use table-driven tests for testing multiple scenarios
- Test both success and error conditions
- Keep tests independent and deterministic
- Use descriptive test names that explain what is being tested

## Code Style

### Go Conventions
- Follow standard Go formatting: `go fmt ./...`
- Use meaningful variable and function names
- Keep functions small and focused on a single task

### Comments
- **Minimize comments**: Write self-documenting code with clear variable and function names
- **Only comment when necessary**: Add comments for particularly tricky code where the intent might be unclear
- **Avoid obvious comments**: Don't comment what the code obviously does (e.g., `// Create a new client`)
- **Focus on why, not what**: When commenting, explain the reasoning behind complex logic, not just what it does
- **Examples of good comments**:
  - Complex algorithms or business logic
  - Non-obvious error handling strategies
  - Performance-critical code sections
  - Workarounds for library limitations
- **Examples of unnecessary comments**:
  - Function signatures that repeat the function name
  - Obvious variable assignments
  - Simple control flow structures

### Error Handling
- Always handle errors explicitly
- Provide context in error messages
- Use the logging package for structured error reporting
- Prefer returning errors over panicking

### Dependencies
- Minimize external dependencies
- Prefer standard library solutions when possible
- Keep package interfaces clean and focused
- Use dependency injection for testability
