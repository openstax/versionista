package main

import (
	"strings"
	"testing"
	"github.com/spf13/viper"
)

// Test helper to verify repo config parsing without making API calls
func TestProjectConfigParsing(t *testing.T) {
	// Setup viper config for project
	projectConfig := []RepoConfig{
		{
			Repo:      "org1/repo1",
			Alias:     "First Repo",
			Jira:      true,
			CrossLink: true,
		},
		{
			Repo:      "org2/repo2",
			Alias:     "Second Repo",
			Jira:      false,
			CrossLink: false,
		},
	}
	
	viper.Set("projects.testproject", projectConfig)
	defer viper.Set("projects.testproject", nil)

	// Test that we can parse the project config correctly
	var parsedConfigs []RepoConfig
	err := viper.UnmarshalKey("projects.testproject", &parsedConfigs)
	
	if err != nil {
		t.Fatalf("Failed to parse project config: %v", err)
	}
	
	if len(parsedConfigs) != 2 {
		t.Errorf("Expected 2 repo configs, got %d", len(parsedConfigs))
	}
	
	// Verify first config
	if parsedConfigs[0].Repo != "org1/repo1" {
		t.Errorf("Expected first repo to be 'org1/repo1', got '%s'", parsedConfigs[0].Repo)
	}
	if parsedConfigs[0].Alias != "First Repo" {
		t.Errorf("Expected first alias to be 'First Repo', got '%s'", parsedConfigs[0].Alias)
	}
	if !parsedConfigs[0].Jira {
		t.Errorf("Expected first repo to have JIRA enabled")
	}
	
	// Verify second config
	if parsedConfigs[1].Repo != "org2/repo2" {
		t.Errorf("Expected second repo to be 'org2/repo2', got '%s'", parsedConfigs[1].Repo)
	}
	if parsedConfigs[1].Alias != "Second Repo" {
		t.Errorf("Expected second alias to be 'Second Repo', got '%s'", parsedConfigs[1].Alias)
	}
	if parsedConfigs[1].Jira {
		t.Errorf("Expected second repo to have JIRA disabled")
	}
	
	// Verify CrossLink settings
	if !parsedConfigs[0].CrossLink {
		t.Errorf("Expected first repo to have CrossLink enabled")
	}
	if parsedConfigs[1].CrossLink {
		t.Errorf("Expected second repo to have CrossLink disabled")
	}
}

func TestRepoSpecParsing(t *testing.T) {
	tests := []struct {
		name     string
		repoSpec string
		expected struct {
			isProject    bool
			ownerRepo    []string
		}
	}{
		{
			name:     "direct repo specification",
			repoSpec: "owner/repo",
			expected: struct {
				isProject    bool
				ownerRepo    []string
			}{false, []string{"owner", "repo"}},
		},
		{
			name:     "project specification", 
			repoSpec: "myproject",
			expected: struct {
				isProject    bool
				ownerRepo    []string
			}{true, []string{"myproject"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ownerRepo := strings.Split(tt.repoSpec, "/")
			isProject := len(ownerRepo) == 1

			if isProject != tt.expected.isProject {
				t.Errorf("Expected isProject to be %v, got %v", tt.expected.isProject, isProject)
			}
			
			if len(ownerRepo) != len(tt.expected.ownerRepo) {
				t.Errorf("Expected ownerRepo length %d, got %d", len(tt.expected.ownerRepo), len(ownerRepo))
			}
			
			for i, expected := range tt.expected.ownerRepo {
				if ownerRepo[i] != expected {
					t.Errorf("Expected ownerRepo[%d] to be '%s', got '%s'", i, expected, ownerRepo[i])
				}
			}
		})
	}
}