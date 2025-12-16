package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	GHToken    string                        `mapstructure:"gh_token"`
	Projects   map[string][]RepoConfig       `mapstructure:"projects"`
	JiraBoards []string                      `mapstructure:"jira_boards"`
	JiraOrgId  string                        `mapstructure:"jira_org_id"`
	Branches   map[string]string             `mapstructure:"branches"`
}

type RepoConfig struct {
	Repo      string `mapstructure:"repo"`
	Alias     string `mapstructure:"alias"`
	Jira      bool   `mapstructure:"jira"`
	CrossLink bool   `mapstructure:"crossLink"`
}


func LoadFromPath(configPath string) (*Config, error) {
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.SetConfigName(".versionista")
		viper.SetConfigType("yaml")
		
		if homeDir, err := os.UserHomeDir(); err == nil {
			viper.AddConfigPath(homeDir)
		}
		
		if cwd, err := os.Getwd(); err == nil {
			viper.AddConfigPath(cwd)
		}
	}
	
	if err := viper.ReadInConfig(); err != nil {
		if configPath != "" {
			return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}
		return nil, fmt.Errorf("failed to read config file .versionista.yml from home directory or current directory: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) GetProjectRepos(projectName string) ([]RepoConfig, error) {
	repos, exists := c.Projects[projectName]
	if !exists {
		return nil, fmt.Errorf("project %s not found in configuration", projectName)
	}
	return repos, nil
}

func (c *Config) GetBranch(repoSpec string) string {
	if branch, exists := c.Branches[repoSpec]; exists {
		return branch
	}
	return "main"
}

// FindProjectByRepository returns the project name that contains the given repository.
// The repoName can be either the short name (e.g., "my-repo") or the full name (e.g., "owner/my-repo").
func (c *Config) FindProjectByRepository(repoName string) (string, error) {
	var matchingProjects []string

	for projectName, repos := range c.Projects {
		for _, repo := range repos {
			// Check if it matches the full repo spec or just the repo name part
			if repo.Repo == repoName {
				matchingProjects = append(matchingProjects, projectName)
				break
			}
			// Extract just the repo name from "owner/repo" format
			parts := strings.Split(repo.Repo, "/")
			if len(parts) == 2 && parts[1] == repoName {
				matchingProjects = append(matchingProjects, projectName)
				break
			}
		}
	}

	if len(matchingProjects) == 0 {
		return "", fmt.Errorf("repository '%s' not found in any project", repoName)
	}
	if len(matchingProjects) > 1 {
		return "", fmt.Errorf("repository '%s' found in multiple projects (%v), please specify one using --project flag", repoName, matchingProjects)
	}
	return matchingProjects[0], nil
}

func (c *Config) GetProjectName(providedProject string, args []string) (string, error) {
	// If project flag is provided, use it
	if providedProject != "" {
		if _, exists := c.Projects[providedProject]; !exists {
			return "", fmt.Errorf("project '%s' not found in configuration", providedProject)
		}
		return providedProject, nil
	}
	
	// If project name is provided as argument, use it
	if len(args) > 0 {
		projectName := args[0]
		if _, exists := c.Projects[projectName]; !exists {
			return "", fmt.Errorf("project '%s' not found in configuration", projectName)
		}
		return projectName, nil
	}

	// If no project name provided, check if there's only one project
	if len(c.Projects) == 1 {
		for projectName := range c.Projects {
			return projectName, nil
		}
	}

	// Multiple projects exist, require explicit specification
	var projectNames []string
	for name := range c.Projects {
		projectNames = append(projectNames, name)
	}
	return "", fmt.Errorf("multiple projects found (%v), please specify one using --project flag or as argument", projectNames)
}

func (c *Config) Validate() error {
	if c.GHToken == "" {
		return fmt.Errorf("gh_token is required in configuration")
	}

	if len(c.Projects) == 0 {
		return fmt.Errorf("at least one project must be configured")
	}

	jiraEnabledProjectFound := false
	for projectName, repos := range c.Projects {
		if len(repos) == 0 {
			return fmt.Errorf("project %s has no repositories configured", projectName)
		}

		for i, repo := range repos {
			if repo.Repo == "" {
				return fmt.Errorf("project %s, repo %d: repo field is required", projectName, i)
			}
			if repo.Jira {
				jiraEnabledProjectFound = true
			}
		}
	}

	if jiraEnabledProjectFound && c.JiraOrgId == "" {
		return fmt.Errorf("jira_org_id is required when a project has jira enabled")
	}

	return nil
}