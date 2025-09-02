package main

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	configContent := `gh_token: test_token
projects:
  testproject:
    - repo: org1/repo1
      alias: Test Repo 1
      jira: true
      crossLink: true
    - repo: org2/repo2
      alias: Test Repo 2
      jira: false
      crossLink: false

jira_boards:
  - TEST
  - PROJ

jira_org_id: my-org

branches:
  org1/repo1: main
  org2/repo2: develop`

	// Create temporary directory and config file
	tmpDir, err := ioutil.TempDir("", "versionista_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := tmpDir + "/.versionista.yml"
	if err := ioutil.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test loading from specific path
	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Validate loaded config
	if cfg.GHToken != "test_token" {
		t.Errorf("Expected gh_token 'test_token', got: %s", cfg.GHToken)
	}

	if len(cfg.Projects) != 1 {
		t.Errorf("Expected 1 project, got: %d", len(cfg.Projects))
	}

	testProject, exists := cfg.Projects["testproject"]
	if !exists {
		t.Fatal("Expected 'testproject' to exist")
	}

	if len(testProject) != 2 {
		t.Errorf("Expected 2 repositories in testproject, got: %d", len(testProject))
	}

	// Test first repo config
	repo1 := testProject[0]
	if repo1.Repo != "org1/repo1" {
		t.Errorf("Expected repo 'org1/repo1', got: %s", repo1.Repo)
	}
	if repo1.Alias != "Test Repo 1" {
		t.Errorf("Expected alias 'Test Repo 1', got: %s", repo1.Alias)
	}
	if !repo1.Jira {
		t.Error("Expected Jira to be enabled")
	}
	if !repo1.CrossLink {
		t.Error("Expected CrossLink to be enabled")
	}

	if cfg.JiraOrgId != "my-org" {
		t.Errorf("Expected jira_org_id 'my-org', got: %s", cfg.JiraOrgId)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "valid config with jira",
			config: Config{
				GHToken:   "test_token",
				JiraOrgId: "my-org",
				Projects: map[string][]RepoConfig{
					"test": {
						{Repo: "owner/repo", Alias: "Test", Jira: true, CrossLink: false},
					},
				},
			},
			expectError: false,
		},
		{
			name: "jira enabled without jira_org_id",
			config: Config{
				GHToken: "test_token",
				Projects: map[string][]RepoConfig{
					"test": {
						{Repo: "owner/repo", Alias: "Test", Jira: true, CrossLink: false},
					},
				},
			},
			expectError: true,
		},
		{
			name: "missing gh_token",
			config: Config{
				Projects: map[string][]RepoConfig{
					"test": {
						{Repo: "owner/repo", Alias: "Test", Jira: true, CrossLink: false},
					},
				},
			},
			expectError: true,
		},
		{
			name: "no projects",
			config: Config{
				GHToken:  "test_token",
				Projects: map[string][]RepoConfig{},
			},
			expectError: true,
		},
		{
			name: "empty repo name",
			config: Config{
				GHToken: "test_token",
				Projects: map[string][]RepoConfig{
					"test": {
						{Repo: "", Alias: "Test", Jira: true, CrossLink: false},
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestGetProjectRepos(t *testing.T) {
	cfg := &Config{
		Projects: map[string][]RepoConfig{
			"project1": {
				{Repo: "org1/repo1", Alias: "Repo 1", Jira: true, CrossLink: false},
				{Repo: "org1/repo2", Alias: "Repo 2", Jira: false, CrossLink: true},
			},
			"project2": {
				{Repo: "org2/repo1", Alias: "Other Repo", Jira: true, CrossLink: false},
			},
		},
	}

	// Test existing project
	repos, err := cfg.GetProjectRepos("project1")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(repos) != 2 {
		t.Errorf("Expected 2 repos, got: %d", len(repos))
	}

	// Test non-existing project
	_, err = cfg.GetProjectRepos("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent project")
	}
}

func TestGetBranch(t *testing.T) {
	cfg := &Config{
		Branches: map[string]string{
			"org/repo1": "develop",
			"org/repo2": "feature-branch",
		},
	}

	// Test existing branch mapping
	branch := cfg.GetBranch("org/repo1")
	if branch != "develop" {
		t.Errorf("Expected 'develop', got: %s", branch)
	}

	// Test default branch for unmapped repo
	branch = cfg.GetBranch("org/unmapped")
	if branch != "main" {
		t.Errorf("Expected 'main' (default), got: %s", branch)
	}
}