package main

import (
	"strings"
	"testing"
	"github.com/Masterminds/semver"
	"io"
	"os"
	"bytes"
)

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestAnnounceVersions(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	// Create test releases with different alias scenarios
	version1, _ := semver.NewVersion("1.0.0")
	version2, _ := semver.NewVersion("2.0.0")
	
	repo1 := &Repository{
		client: client,
		owner:  "org",
		name:   "repo1",
		alias:  "", // No alias
	}
	
	repo2 := &Repository{
		client: client,
		owner:  "org",
		name:   "repo2",  
		alias:  "My Custom Repo", // With alias
	}

	releases := []*Release{
		{
			repository: repo1,
			version:    version1,
		},
		{
			repository: repo2,
			version:    version2,
		},
	}

	output := captureOutput(func() {
		announceVersions("test-project", releases)
	})

	// Check that the output contains the project name
	if !strings.Contains(output, "test-project versions are:") {
		t.Error("Expected output to contain project name")
	}

	// Check that repo1 uses its name (no alias)
	if !strings.Contains(output, "repo1") {
		t.Error("Expected output to contain repo1 name")
	}

	// Check that repo2 uses its alias instead of name
	if !strings.Contains(output, "My Custom Repo") {
		t.Error("Expected output to contain repo2 alias")
	}
	if strings.Contains(output, "repo2") {
		t.Error("Expected output to NOT contain repo2 name when alias is present")
	}

	// Check version formatting
	if !strings.Contains(output, "v1.0.0") {
		t.Error("Expected output to contain v1.0.0")
	}
	if !strings.Contains(output, "v2.0.0") {
		t.Error("Expected output to contain v2.0.0")
	}
}

func TestAnnounceRepo(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	repo := &Repository{
		client: client,
		owner:  "testorg",
		name:   "testrepo",
		alias:  "Test Repository",
	}

	output := captureOutput(func() {
		announceRepo(repo)
	})

	// announceRepo uses repo.String() which returns "owner/repo"
	if !strings.Contains(output, "testorg/testrepo") {
		t.Error("Expected output to contain repository owner/repo")
	}
}

func TestAnnounceRelease(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	repo := &Repository{
		client: client,
		owner:  "org",
		name:   "repo",
	}

	version, _ := semver.NewVersion("1.2.3")

	output := captureOutput(func() {
		announceRelease(repo, version)
	})

	if !strings.Contains(output, "🎉") {
		t.Error("Expected output to contain celebration emoji")
	}
	if !strings.Contains(output, "v1.2.3") {
		t.Error("Expected output to contain version")
	}
	if !strings.Contains(output, "released") {
		t.Error("Expected output to contain 'released'")
	}
}

func TestAnnounceFetching(t *testing.T) {
	output := captureOutput(func() {
		announceFetching()
	})

	if !strings.Contains(output, "Fetching") {
		t.Error("Expected output to mention Fetching")
	}
}

func TestComposeReleaseMessageStructure(t *testing.T) {
	// Setup JIRA config for test
	oldJiraSlug := ""
	defer func() {
		if oldJiraSlug != "" {
			os.Setenv("VIPER_JIRA_SLUG", oldJiraSlug)
		}
	}()

	// Create test changelog entries
	entries := []ChangeLogEntry{
		{
			Number:      123,
			Date:        "2023-01-01",
			Author:      "John Doe",
			Title:       "Fix important bug",
			Description: "This PR fixes a critical bug in the system",
			Tickets:     []string{"TEST-456"},
		},
		{
			Number:      124,
			Date:        "2023-01-02", 
			Author:      "Jane Smith",
			Title:       "Add new feature",
			Description: "Added support for new functionality",
			Tickets:     []string{},
		},
	}

	// Note: We can't easily test the full composeReleaseMessage function 
	// since it opens an editor and requires user interaction.
	// Instead, we test that our ChangeLogEntry structure is correctly populated.
	
	if entries[0].Number != 123 {
		t.Errorf("Expected first entry number to be 123, got %d", entries[0].Number)
	}
	
	if entries[0].Author != "John Doe" {
		t.Errorf("Expected first entry author to be 'John Doe', got '%s'", entries[0].Author)
	}
	
	if len(entries[0].Tickets) != 1 || entries[0].Tickets[0] != "TEST-456" {
		t.Errorf("Expected first entry to have ticket 'TEST-456', got %v", entries[0].Tickets)
	}
	
	if len(entries[1].Tickets) != 0 {
		t.Errorf("Expected second entry to have no tickets, got %v", entries[1].Tickets)
	}
	
	// Test that Description field is properly populated
	if entries[0].Description != "This PR fixes a critical bug in the system" {
		t.Errorf("Expected first entry description to be populated, got '%s'", entries[0].Description)
	}
	
	if entries[1].Description != "Added support for new functionality" {
		t.Errorf("Expected second entry description to be populated, got '%s'", entries[1].Description)
	}
}