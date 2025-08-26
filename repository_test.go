package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"github.com/spf13/viper"
)

func TestGetLatestRelease(t *testing.T) {
	client, mux, _, teardown := setup()
	config := RepoConfig{
		Repo:      "foo/bar",
		Alias:     "",
		Jira:      false,
		CrossLink: false,
	}
	repo := NewRepository(config, client)
	defer teardown()
	specifiedVersion := "1.1.42"

	mux.HandleFunc("/repos/foo/bar/releases", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `[{"id":1, "tag_name": "v1.1.42"}]`)
	})
	mux.HandleFunc("/repos/foo/bar/compare/v1.1.42...master", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
	})
	repo.resolveVersions(context.Background())
	parsedVersion := repo.latestRelease
	if parsedVersion.String() != specifiedVersion {
		t.Errorf("Latest release is %s, wanted %s", parsedVersion.String(), specifiedVersion)
	}
}

func TestNewRepositoryConfig(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	tests := []struct {
		name     string
		config   RepoConfig
		expected struct {
			owner            string
			name             string
			alias            string
			jiraEnabled      bool
			crossLinkEnabled bool
		}
	}{
		{
			name: "basic config",
			config: RepoConfig{
				Repo:      "owner/repo",
				Alias:     "",
				Jira:      false,
				CrossLink: false,
			},
			expected: struct {
				owner            string
				name             string
				alias            string
				jiraEnabled      bool
				crossLinkEnabled bool
			}{"owner", "repo", "", false, false},
		},
		{
			name: "config with alias and jira enabled",
			config: RepoConfig{
				Repo:      "myorg/myrepo",
				Alias:     "My Custom Name",
				Jira:      true,
				CrossLink: false,
			},
			expected: struct {
				owner            string
				name             string
				alias            string
				jiraEnabled      bool
				crossLinkEnabled bool
			}{"myorg", "myrepo", "My Custom Name", true, false},
		},
		{
			name: "config with crossLink enabled",
			config: RepoConfig{
				Repo:      "org/crosslinked-repo",
				Alias:     "Cross-Linked Repo",
				Jira:      false,
				CrossLink: true,
			},
			expected: struct {
				owner            string
				name             string
				alias            string
				jiraEnabled      bool
				crossLinkEnabled bool
			}{"org", "crosslinked-repo", "Cross-Linked Repo", false, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewRepository(tt.config, client)

			if repo.owner != tt.expected.owner {
				t.Errorf("Expected owner %s, got %s", tt.expected.owner, repo.owner)
			}
			if repo.name != tt.expected.name {
				t.Errorf("Expected name %s, got %s", tt.expected.name, repo.name)
			}
			if repo.alias != tt.expected.alias {
				t.Errorf("Expected alias %s, got %s", tt.expected.alias, repo.alias)
			}
			if repo.jiraEnabled != tt.expected.jiraEnabled {
				t.Errorf("Expected jiraEnabled %v, got %v", tt.expected.jiraEnabled, repo.jiraEnabled)
			}
			if repo.crossLinkEnabled != tt.expected.crossLinkEnabled {
				t.Errorf("Expected crossLinkEnabled %v, got %v", tt.expected.crossLinkEnabled, repo.crossLinkEnabled)
			}
		})
	}
}

func TestNewRepositoryWithBranches(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	// Setup viper config for branches
	viper.Set("branches.owner/repo", "develop")
	defer viper.Set("branches.owner/repo", nil)

	config := RepoConfig{
		Repo:      "owner/repo",
		Alias:     "",
		Jira:      false,
		CrossLink: false,
	}
	repo := NewRepository(config, client)

	if repo.commitSHA != "develop" {
		t.Errorf("Expected commitSHA 'develop', got '%s'", repo.commitSHA)
	}
}

func TestNewRepositoryWithJiraBoards(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	// Setup viper config for jira boards
	viper.Set("jira_boards", []string{"TEST", "PROJ"})
	defer viper.Set("jira_boards", nil)

	config := RepoConfig{
		Repo:  "owner/repo",
		Alias: "",
		Jira:  true,
	}
	repo := NewRepository(config, client)

	if repo.ticketMatcher == nil {
		t.Error("Expected ticketMatcher to be created when JIRA is enabled and boards are configured")
	}

	// Test that the matcher works
	if !repo.ticketMatcher.MatchString("This fixes TEST-123") {
		t.Error("Expected ticketMatcher to match 'TEST-123'")
	}
	if !repo.ticketMatcher.MatchString("PROJ-456 was fixed") {
		t.Error("Expected ticketMatcher to match 'PROJ-456'")
	}
}

func TestNewRepositoryJiraDisabled(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	// Setup viper config for jira boards
	viper.Set("jira_boards", []string{"TEST", "PROJ"})
	defer viper.Set("jira_boards", nil)

	config := RepoConfig{
		Repo:  "owner/repo",
		Alias: "",
		Jira:  false, // JIRA disabled
	}
	repo := NewRepository(config, client)

	if repo.ticketMatcher != nil {
		t.Error("Expected ticketMatcher to be nil when JIRA is disabled")
	}
}

func TestRepositoryString(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	config := RepoConfig{
		Repo:  "testorg/testrepo",
		Alias: "Test Repo",
		Jira:  false,
	}
	repo := NewRepository(config, client)

	expected := "testorg/testrepo"
	if repo.String() != expected {
		t.Errorf("Expected String() to return %s, got %s", expected, repo.String())
	}
}

func TestAppendMatches(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	// Setup viper config for jira boards
	viper.Set("jira_boards", []string{"TEST", "PROJ"})
	defer viper.Set("jira_boards", nil)

	tests := []struct {
		name        string
		jiraEnabled bool
		content     string
		expected    []string
	}{
		{
			name:        "jira disabled",
			jiraEnabled: false,
			content:     "This fixes TEST-123",
			expected:    []string{},
		},
		{
			name:        "jira enabled with match",
			jiraEnabled: true,
			content:     "This fixes TEST-123",
			expected:    []string{"TEST-123"},
		},
		{
			name:        "jira enabled with multiple matches",
			jiraEnabled: true,
			content:     "This fixes TEST-123 and PROJ-456",
			expected:    []string{"TEST-123"}, // appendMatches only finds first match
		},
		{
			name:        "jira enabled no match",
			jiraEnabled: true,
			content:     "No ticket references here",
			expected:    []string{},
		},
		{
			name:        "jira enabled with existing matches",
			jiraEnabled: true,
			content:     "This fixes TEST-789",
			expected:    []string{"existing", "TEST-789"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := RepoConfig{
				Repo:  "owner/repo",
				Alias: "",
				Jira:  tt.jiraEnabled,
			}
			repo := NewRepository(config, client)

			initialMatches := []string{}
			if tt.name == "jira enabled with existing matches" {
				initialMatches = []string{"existing"}
			}

			result := repo.appendMatches(initialMatches, tt.content)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d matches, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected match[%d] to be %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}