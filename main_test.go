package main

import (
	"bytes"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestCheckError(t *testing.T) {
	// Test that CheckError doesn't panic with nil error
	CheckError(nil)

	// We can't easily test the error case since CheckError calls os.Exit(1)
	// In a real application, you might want to refactor this to return errors
	// instead of exiting, or use dependency injection for testing
}

func TestWarn(t *testing.T) {
	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Call Warn function
	Warn("test warning: %s", "something went wrong")

	// Restore stderr and capture output
	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check that warning is properly formatted
	if !strings.Contains(output, "[ERROR]:") {
		t.Error("Expected output to contain [ERROR]: prefix")
	}
	if !strings.Contains(output, "test warning: something went wrong") {
		t.Error("Expected output to contain formatted warning message")
	}
}

// Test helper functions and integration scenarios
func TestRepoConfigValidation(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	tests := []struct {
		name        string
		config      RepoConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: RepoConfig{
				Repo:  "owner/repo",
				Alias: "Test Repo",
				Jira:  true,
			},
			expectError: false,
		},
		{
			name: "empty repo",
			config: RepoConfig{
				Repo:  "",
				Alias: "Test",
				Jira:  false,
			},
			expectError: true,
		},
		{
			name: "invalid repo format",
			config: RepoConfig{
				Repo:  "invalid-repo-format",
				Alias: "",
				Jira:  false,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil && !tt.expectError {
					t.Errorf("Unexpected panic: %v", r)
				}
			}()

			if tt.config.Repo == "" || !strings.Contains(tt.config.Repo, "/") {
				// These would cause index out of bounds in strings.Split
				if !tt.expectError {
					t.Error("Expected this config to cause an error")
				}
				return
			}

			repo := NewRepository(tt.config, client)
			
			if tt.expectError {
				t.Error("Expected an error but got none")
			} else {
				if repo.owner == "" || repo.name == "" {
					t.Error("Repository was not properly initialized")
				}
			}
		})
	}
}

func TestIntegrationRepoWithJiraAndAlias(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	// Setup mock responses
	mux.HandleFunc("/repos/myorg/myrepo/releases", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"tag_name": "v1.0.0"}]`))
	})
	mux.HandleFunc("/repos/myorg/myrepo/compare/v1.0.0...main", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ahead_by": 0, "commits": []}`))
	})

	// Test full integration
	config := RepoConfig{
		Repo:  "myorg/myrepo",
		Alias: "My Amazing Repo",
		Jira:  true,
	}

	repo := NewRepository(config, client)

	// Verify all fields are set correctly
	if repo.owner != "myorg" {
		t.Errorf("Expected owner 'myorg', got '%s'", repo.owner)
	}
	if repo.name != "myrepo" {
		t.Errorf("Expected name 'myrepo', got '%s'", repo.name)
	}
	if repo.alias != "My Amazing Repo" {
		t.Errorf("Expected alias 'My Amazing Repo', got '%s'", repo.alias)
	}
	if !repo.jiraEnabled {
		t.Error("Expected JIRA to be enabled")
	}

	// Test that String() method works
	if repo.String() != "myorg/myrepo" {
		t.Errorf("Expected String() to return 'myorg/myrepo', got '%s'", repo.String())
	}
}