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
}

func NewCLI(cfg *Config, logger *Logger) *CLI {
	client := NewClient(cfg.GHToken)
	manager := NewManager(client, logger, cfg.JiraBoards, cfg.JiraOrgId)

	return &CLI{
		config:  cfg,
		logger:  logger,
		client:  client,
		manager: manager,
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

	// Run the actual function
	err := fn()
	
	// Stop the spinner
	done <- err
	
	// Clear the spinner line by overwriting with spaces and then clearing again
	fmt.Printf("\r%s\r", strings.Repeat(" ", len(message)+2))
	
	return err
}

func (c *CLI) ProcessRepositories(ctx context.Context, repoSpec string) ([]*ReleaseRepository, error) {
	ownerRepo := strings.Split(repoSpec, "/")
	var repoConfigs []RepoConfig

	if len(ownerRepo) == 1 {
		configs, err := c.config.GetProjectRepos(repoSpec)
		if err != nil {
			return nil, err
		}
		repoConfigs = configs
	} else {
		repoConfigs = []RepoConfig{
			{
				Repo:      repoSpec,
				Alias:     "",
				Jira:      true,
				CrossLink: false,
			},
		}
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

	err := c.runWithSpinner("Fetching repository information...", func() error {
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

func (c *CLI) getProjectName(args []string) (string, error) {
	// If project name is provided, use it
	if len(args) > 0 {
		return args[0], nil
	}

	// If no project name provided, check if there's only one project
	if len(c.config.Projects) == 1 {
		for projectName := range c.config.Projects {
			c.logger.Info("Using default project: %s", projectName)
			return projectName, nil
		}
	}

	// Multiple projects exist, require explicit specification
	var projectNames []string
	for name := range c.config.Projects {
		projectNames = append(projectNames, name)
	}
	return "", fmt.Errorf("multiple projects found (%v), please specify one: versionista [project-name]", projectNames)
}

func (c *CLI) releaseCommand(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	
	projectName, err := c.getProjectName(args)
	if err != nil {
		c.logger.FatalErr(err, "Failed to determine project")
	}
	
	repos, err := c.ProcessRepositories(ctx, projectName)
	if err != nil {
		c.logger.FatalErr(err, "Failed to process repositories")
	}

	releaseType := TypeRegular
	// Everything is interactive now
	interactive := true

	// For interactive mode with multiple repos, show proposed changelogs and let user select
	if interactive && len(repos) > 1 {
		// Generate proposed changelogs for all repos with spinner
		var proposedChangelogs map[*ReleaseRepository]*ProposedRelease
		err := c.runWithSpinner("Fetching proposed changelogs for all repositories...", func() error {
			proposedChangelogs = make(map[*ReleaseRepository]*ProposedRelease)
			for _, repo := range repos {
				proposed, err := c.manager.GenerateProposedRelease(ctx, repo)
				if err != nil {
					c.logger.Error("Failed to generate proposed release for %s: %v", repo.Repository, err)
					continue
				}
				proposedChangelogs[repo] = proposed
			}
			return nil
		})
		if err != nil {
			c.logger.FatalErr(err, "Failed to fetch proposed changelogs")
		}

		// Show all proposed changelogs and let user select
		selectedRepos, err := PromptForProjectSelection(proposedChangelogs)
		if err != nil {
			c.logger.FatalErr(err, "Failed to get project selection")
		}

		if len(selectedRepos) == 0 {
			c.logger.Info("No repositories selected for release")
			return
		}

		// Process releases for selected repositories with crossLinks
		releases, err := c.manager.ProcessSelectedReleases(ctx, selectedRepos, releaseType, repos)
		if err != nil {
			c.logger.FatalErr(err, "Failed to process selected releases")
		}

		c.logger.Info("Release processing completed for %s", projectName)
		for _, rel := range releases {
			c.logger.Info("- %s: %s", rel.Repository.Repository, FormatVersion(rel.Version))
		}
		return
	}

	// Original flow for single repo
	var releases []*Release
	for _, repo := range repos {
		var entries []Entry
		var err error
		
		// Run preparation with spinner
		message := fmt.Sprintf("Processing release for %s", repo.Repository.String())
		err = c.runWithSpinner(message, func() error {
			entries, err = c.manager.PrepareReleaseData(ctx, repo)
			return err
		})
		
		if err != nil {
			c.logger.Error("Failed to prepare release data for %s: %v", repo.Repository, err)
			continue
		}
		
		// Now run the interactive part without spinner
		rel, err := c.manager.ProcessReleaseInteractiveWithEntries(ctx, repo, releaseType, repos, entries)
		if err != nil {
			c.logger.Error("Failed to process release for %s: %v", repo.Repository, err)
			continue
		}
		
		releases = append(releases, rel)
	}

	c.logger.Info("Release processing completed for %s", projectName)
	for _, rel := range releases {
		c.logger.Info("- %s: %s", rel.Repository.Repository, FormatVersion(rel.Version))
	}
}

func (c *CLI) reviewCommand(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	
	projectName, err := c.getProjectName(args)
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

func configureCliCommands() {
	var configPath string
	var logLevel string

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

		return NewCLI(cfg, logger)
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
			cli.releaseCommand(cmd, args)
		},
	}

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to configuration file (default: ~/.versionista.yml or ./.versionista.yml)")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "warn", "Set logging level (debug, info, warn, error)")

	releaseCmd := &cobra.Command{
		Use:   "release [project-name|owner/repo]",
		Short: "Create releases for project(s) or specific repository",
		Args:  cobra.MaximumNArgs(1), // Allow 0 or 1 arguments
		Run: func(cmd *cobra.Command, args []string) {
			cli := loadConfigAndCreateCLI()
			cli.releaseCommand(cmd, args)
		},
	}

	reviewCmd := &cobra.Command{
		Use:   "review [project-name|owner/repo]",
		Short: "Review latest versions of project(s) or specific repository",
		Args:  cobra.MaximumNArgs(1), // Allow 0 or 1 arguments
		Run: func(cmd *cobra.Command, args []string) {
			cli := loadConfigAndCreateCLI()
			cli.reviewCommand(cmd, args)
		},
	}

	rootCmd.AddCommand(releaseCmd)
	rootCmd.AddCommand(reviewCmd)

	if err := rootCmd.Execute(); err != nil {
		// Create a basic logger for command execution errors
		logger := NewLogger()
		logger.FatalErr(err, "Command execution failed")
	}
}
