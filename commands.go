package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

type CLI struct {
	config  *Config
	logger  *Logger
	client  *Client
	manager *Manager
	dryRun  bool
}

func NewCLI(cfg *Config, logger *Logger, dryRun bool) *CLI {
	client := NewClient(cfg.GHToken)
	manager := NewManager(client, logger, cfg.JiraBoards, cfg.JiraOrgId, dryRun)

	return &CLI{
		config:  cfg,
		logger:  logger,
		client:  client,
		manager: manager,
		dryRun:  dryRun,
	}
}

func (c *CLI) runWithSpinner(message string, fn func() error) error {
	// Don't show spinner if debug level is enabled
	if c.logger.IsDebugEnabled() {
		c.logger.Debug("Starting: %s", message)
		return fn()
	}

	// Simple spinner characters
	spinChars := []string{"|", "/", "-", "\\"}

	// Channel to signal completion
	done := make(chan error, 1)

	// Start the spinner in a goroutine
	go func() {
		for i := 0; ; i++ {
			select {
			case <-done:
				return
			default:
				fmt.Printf("\r%s %s", spinChars[i%len(spinChars)], message)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
	err := fn()
	done <- err
	// Clear the spinner line by overwriting with spaces and then clearing again
	fmt.Printf("\r%s\r", strings.Repeat(" ", len(message)+2))

	return err
}

func (c *CLI) ProcessRepositories(ctx context.Context, repoSpec string) ([]*ReleaseRepository, error) {
	repoConfigs, err := c.config.GetProjectRepos(repoSpec)
	if err != nil {
		return nil, err
	}

	var repos []*ReleaseRepository
	for _, cfg := range repoConfigs {
		ghRepo, err := ParseRepoSpec(cfg.Repo)
		if err != nil {
			return nil, err
		}

		commitSHA := c.config.GetBranch(cfg.Repo)
		repo := NewRepository(ghRepo, cfg.Alias, cfg.Jira, cfg.CrossLink, commitSHA)
		repos = append(repos, repo)
	}

	err = c.runWithSpinner("Fetching repository information...", func() error {
		wg := sync.WaitGroup{}
		var fetchError error
		for _, repo := range repos {
			wg.Add(1)
			go func(r *ReleaseRepository) {
				defer wg.Done()
				if err := c.manager.ResolveVersions(ctx, r); err != nil {
					c.logger.Error("Failed to resolve versions for %s: %v", r.Repository, err)
					fetchError = err
				}
			}(repo)
		}
		wg.Wait()
		return fetchError
	})

	if err != nil {
		return nil, err
	}

	return repos, nil
}


func (c *CLI) releaseCommand(args []string, providedProject string) {
	ctx := context.Background()

	projectName, err := c.config.GetProjectName(providedProject, args)
	if err != nil {
		c.logger.FatalErr(err, "Failed to determine project")
	}

	repos, err := c.ProcessRepositories(ctx, projectName)
	if err != nil {
		c.logger.FatalErr(err, "Failed to process repositories")
	}

	releaseType := TypeRegular

	var releases []*Release

	for _, repo := range repos {
		var entries []Entry
		err := c.runWithSpinner(fmt.Sprintf("Fetching changelog for %s...", repo.GetDisplayName()), func() error {
			var err error
			entries, err = c.manager.GenerateChangelog(ctx, repo)
			return err
		})
		if err != nil {
			c.logger.FatalErr(err, fmt.Sprintf("Failed to generate changelog for %s", repo.Repository))
		}

		// Process this repository individually using the interactive method
		release, err := c.manager.ProcessReleaseInteractiveWithEntries(ctx, repo, releaseType, repos, entries)
		if err != nil {
			c.logger.FatalErr(err, fmt.Sprintf("Failed to process release for %s", repo.Repository))
		}

		// Only add to releases if it's not a skip (version unchanged means skip)
		if release.Version.String() != repo.LatestRelease.String() {
			releases = append(releases, release)
		}
	}

	c.logger.Info("Release processing completed for %s", projectName)
	for _, rel := range releases {
		c.logger.Info("- %s: %s", rel.Repository.Repository, FormatVersion(rel.Version))
	}
	return

}

func (c *CLI) reviewCommand(args []string, providedProject string) {
	ctx := context.Background()

	projectName, err := c.config.GetProjectName(providedProject, args)
	if err != nil {
		c.logger.FatalErr(err, "Failed to determine project")
	}

	repos, err := c.ProcessRepositories(ctx, projectName)
	if err != nil {
		c.logger.FatalErr(err, "Failed to process repositories")
	}

	c.logger.Info("Latest versions for %s:", projectName)
	for _, repo := range repos {
		displayName := repo.GetDisplayName()
		if repo.LatestRelease != nil {
			c.logger.Info("- %s: %s", displayName, FormatVersion(repo.LatestRelease))
		} else {
			c.logger.Info("- %s: No releases found", displayName)
		}
	}
}

func (c *CLI) hotfixCommand(args []string, providedProject string) {
	ctx := context.Background()

	projectName, err := c.config.GetProjectName(providedProject, args)
	if err != nil {
		c.logger.FatalErr(err, "Failed to determine project")
	}
	repositoryName := args[0]
	sha := args[1]
	
	allRepos, err := c.ProcessRepositories(ctx, projectName)
	if err != nil {
		c.logger.FatalErr(err, "Failed to process repository")
	}
	// Find the repository that matches repositoryName
	var repo *ReleaseRepository
	for _, r := range allRepos {
		if r.Name == repositoryName {
			repo = r
			break
		}
	}
	if repo == nil {
		c.logger.FatalErr(fmt.Errorf("repository '%s' not found in project '%s'", repositoryName, projectName), "Repository not found")
	}

	suffix, err := PromptForHotfixSuffix(repo.LatestRelease, sha)
	if err != nil {
		c.logger.FatalErr(err, "Failed to get hotfix suffix")
	}
	
	// Create hotfix version
	hotfixVersion, err := CreateHotfixVersion(repo.LatestRelease, suffix)
	if err != nil {
		c.logger.FatalErr(err, "Failed to create hotfix version")
	}


	var entries []Entry
	err = c.runWithSpinner(fmt.Sprintf("Generating changelog for hotfix from SHA %s...", sha), func() error {
		entries, err = c.manager.GenerateChangelogFromSHA(ctx, repo, sha)
		return err
	})
	
	if err != nil {
		c.logger.FatalErr(err, "Failed to generate changelog from SHA")
	}
	
	// Build release notes
	var builder strings.Builder
	if len(entries) > 0 {
		builder.WriteString(BuildEntriesTableString(entries, repo.JiraEnabled, c.config.JiraOrgId))
	}
	releaseNotes := builder.String()
	
	// Create the hotfix release
	releaseType := TypeRegular
	if err := c.manager.CreateRelease(ctx, repo, hotfixVersion, releaseNotes, releaseType); err != nil {
		//if err := c.manager.CreateHotfixRelease(ctx, repo, hotfixVersion, releaseNotes, releaseType, sha); err != nil {
		c.logger.FatalErr(err, "Failed to create hotfix release")
	}
	
	c.logger.Info("Hotfix release completed for %s: %s", repo.Repository, FormatVersion(hotfixVersion))
}

func configureCliCommands() {
	var configPath string
	var logLevel string
	var dryRun bool
	var projectName string

	loadConfigAndCreateCLI := func() *CLI {
		level := ParseLevel(logLevel)
		logger := NewLoggerWithLevel(level)

		cfg, err := LoadFromPath(configPath)
		if err != nil {
			logger.FatalErr(err, "Failed to load configuration")
		}

		if err := cfg.Validate(); err != nil {
			logger.FatalErr(err, "Invalid configuration")
		}

		return NewCLI(cfg, logger, dryRun)
	}

	var rootCmd = &cobra.Command{
		Use:   "versionista [project-name|owner/repo]",
		Short: "Create releases for project(s) or specific repository",
		Long: `Versionista automates GitHub releases by analyzing commits and pull requests
since the last release. It supports project-based releases with cross-linking
and individual repository releases.

If no project name is specified and only one project is configured, it will be used automatically.`, 
		Args: cobra.MaximumNArgs(1), // Allow 0 or 1 arguments
		Run: func(cmd *cobra.Command, args []string) {
			cli := loadConfigAndCreateCLI()
			cli.releaseCommand(args, projectName)
		},
	}

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to configuration file (default: ~/.versionista.yml or ./.versionista.yml)")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "warn", "Set logging level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Perform a dry run without creating actual releases")
	rootCmd.PersistentFlags().StringVarP(&projectName, "project", "p", "", "Specify the project to use")

	releaseCmd := &cobra.Command{
		Use:   "release [project-name|owner/repo]",
		Short: "Create releases for project(s) or specific repository",
		Args:  cobra.MaximumNArgs(1), // Allow 0 or 1 arguments
		Run: func(cmd *cobra.Command, args []string) {
			cli := loadConfigAndCreateCLI()
			cli.releaseCommand(args, projectName)
		},
	}

	reviewCmd := &cobra.Command{
		Use:   "review [project-name|owner/repo]",
		Short: "Review latest versions of project(s) or specific repository",
		Args:  cobra.MaximumNArgs(1), // Allow 0 or 1 arguments
		Run: func(cmd *cobra.Command, args []string) {
			cli := loadConfigAndCreateCLI()
			cli.reviewCommand(args, projectName)
		},
	}

	hotfixCmd := &cobra.Command{
		Use:   "hotfix <repository> <sha>",
		Short: "Create a hotfix release for a specific repository from a given SHA",
		Args:  cobra.ExactArgs(2), // Require exactly 2 arguments: repository and SHA
		Run: func(cmd *cobra.Command, args []string) {
			cli := loadConfigAndCreateCLI()
			cli.hotfixCommand(args, projectName)
		},
	}

	rootCmd.AddCommand(releaseCmd)
	rootCmd.AddCommand(reviewCmd)
	rootCmd.AddCommand(hotfixCmd)

	if err := rootCmd.Execute(); err != nil {
		// Create a basic logger for command execution errors
		logger := NewLogger()
		logger.FatalErr(err, "Command execution failed")
	}
}
