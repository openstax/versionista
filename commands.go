package main

import (
	"context"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/openstax/versionista/pkg/config"
	ghclient "github.com/openstax/versionista/pkg/github"
	"github.com/openstax/versionista/pkg/logging"
	"github.com/openstax/versionista/pkg/release"
	"github.com/openstax/versionista/pkg/version"
)

type CLI struct {
	config  *config.Config
	logger  *logging.Logger
	client  *ghclient.Client
	manager *release.Manager
}

func NewCLI(cfg *config.Config, logger *logging.Logger) *CLI {
	client := ghclient.New(cfg.GHToken)
	manager := release.NewManager(client, logger, cfg.JiraBoards)

	return &CLI{
		config:  cfg,
		logger:  logger,
		client:  client,
		manager: manager,
	}
}

func (c *CLI) ProcessRepositories(ctx context.Context, repoSpec string) ([]*release.Repository, error) {
	ownerRepo := strings.Split(repoSpec, "/")
	var repoConfigs []config.RepoConfig

	if len(ownerRepo) == 1 {
		configs, err := c.config.GetProjectRepos(repoSpec)
		if err != nil {
			return nil, err
		}
		repoConfigs = configs
	} else {
		repoConfigs = []config.RepoConfig{
			{
				Repo:      repoSpec,
				Alias:     "",
				Jira:      true,
				CrossLink: false,
			},
		}
	}

	var repos []*release.Repository
	for _, cfg := range repoConfigs {
		ghRepo, err := ghclient.ParseRepoSpec(cfg.Repo)
		if err != nil {
			return nil, err
		}

		commitSHA := c.config.GetBranch(cfg.Repo)
		repo := release.NewRepository(ghRepo, cfg.Alias, cfg.Jira, cfg.CrossLink, commitSHA)
		repos = append(repos, repo)
	}

	c.logger.Info("Fetching repository information...")
	wg := sync.WaitGroup{}
	for _, repo := range repos {
		wg.Add(1)
		go func(r *release.Repository) {
			defer wg.Done()
			if err := c.manager.ResolveVersions(ctx, r); err != nil {
				c.logger.Error("Failed to resolve versions for %s: %v", r.Repository, err)
			}
		}(repo)
	}
	wg.Wait()

	return repos, nil
}

func (c *CLI) releaseCommand(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	
	repos, err := c.ProcessRepositories(ctx, args[0])
	if err != nil {
		c.logger.FatalErr(err, "Failed to process repositories")
	}

	releaseType := release.TypeRegular
	auto, _ := cmd.Flags().GetBool("auto")
	interactive := !auto // Interactive is now the default

	var releases []*release.Release
	for _, repo := range repos {
		c.logger.Info("Processing release for %s", repo.Repository)
		
		var rel *release.Release
		if interactive {
			rel, err = c.manager.ProcessReleaseInteractive(ctx, repo, releaseType, repos)
		} else {
			bumpType := version.BumpPatch
			rel, err = c.manager.ProcessRelease(ctx, repo, releaseType, bumpType, repos)
		}
		
		if err != nil {
			c.logger.Error("Failed to process release for %s: %v", repo.Repository, err)
			continue
		}
		
		releases = append(releases, rel)
	}

	c.logger.Info("Release processing completed for %s", args[0])
	for _, rel := range releases {
		c.logger.Info("- %s: %s", rel.Repository.Repository, version.FormatVersion(rel.Version))
	}
}

func (c *CLI) reviewCommand(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	
	repos, err := c.ProcessRepositories(ctx, args[0])
	if err != nil {
		c.logger.FatalErr(err, "Failed to process repositories")
	}

	c.logger.Info("Latest versions for %s:", args[0])
	for _, repo := range repos {
		displayName := repo.GetDisplayName()
		if repo.LatestRelease != nil {
			c.logger.Info("- %s: %s", displayName, version.FormatVersion(repo.LatestRelease))
		} else {
			c.logger.Info("- %s: No releases found", displayName)
		}
	}
}

func configureCliCommands() {
	var configPath string
	var logLevel string

	var rootCmd = &cobra.Command{
		Use:   "versionista",
		Short: "A simple CLI app to cut releases on GitHub",
		Long: `Versionista automates GitHub releases by analyzing commits and pull requests 
since the last release. It supports project-based releases with cross-linking 
and individual repository releases.`,
	}

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to configuration file (default: ~/.versionista.yml or ./.versionista.yml)")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "warn", "Set logging level (debug, info, warn, error)")

	loadConfigAndCreateCLI := func() *CLI {
		level := logging.ParseLevel(logLevel)
		logger := logging.NewWithLevel(level)
		
		cfg, err := config.LoadFromPath(configPath)
		if err != nil {
			logger.FatalErr(err, "Failed to load configuration")
		}

		if err := cfg.Validate(); err != nil {
			logger.FatalErr(err, "Invalid configuration")
		}

		return NewCLI(cfg, logger)
	}

	releaseCmd := &cobra.Command{
		Use:   "release [project-name|owner/repo]",
		Short: "Create releases for project(s) or specific repository",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cli := loadConfigAndCreateCLI()
			cli.releaseCommand(cmd, args)
		},
	}
	releaseCmd.Flags().BoolP("auto", "a", false, "Use automatic mode with patch bump (default is interactive mode)")

	reviewCmd := &cobra.Command{
		Use:   "review [project-name|owner/repo]",
		Short: "Review latest versions of project(s) or specific repository",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cli := loadConfigAndCreateCLI()
			cli.reviewCommand(cmd, args)
		},
	}

	rootCmd.AddCommand(releaseCmd)
	rootCmd.AddCommand(reviewCmd)

	if err := rootCmd.Execute(); err != nil {
		// Create a basic logger for command execution errors
		logger := logging.New()
		logger.FatalErr(err, "Command execution failed")
	}
}