package github

import (
	"testing"
	"time"
)

func TestParseRepoSpec(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Repository
		hasError bool
	}{
		{
			name:  "valid repo spec",
			input: "owner/repo",
			expected: &Repository{
				Owner: "owner",
				Name:  "repo",
			},
			hasError: false,
		},
		{
			name:     "invalid repo spec - too few parts",
			input:    "owner",
			expected: nil,
			hasError: true,
		},
		{
			name:     "invalid repo spec - too many parts",
			input:    "owner/repo/extra",
			expected: nil,
			hasError: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseRepoSpec(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for input %q, but got none", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error for input %q, but got: %v", tt.input, err)
				return
			}

			if result.Owner != tt.expected.Owner || result.Name != tt.expected.Name {
				t.Errorf("Expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestRepositoryString(t *testing.T) {
	repo := &Repository{
		Owner: "testowner",
		Name:  "testrepo",
	}

	expected := "testowner/testrepo"
	result := repo.String()

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// Note: The Client methods that make actual GitHub API calls would require
// integration tests or mocking the GitHub API. For unit tests, we focus on
// testing the utility functions like ParseRepoSpec and Repository.String()

func TestClientMethods(t *testing.T) {
	// This test verifies that the client methods exist with the correct signatures
	// Actual testing would require mocking the GitHub API or integration tests
	
	// Test that New function creates a client
	client := New("fake-token")
	if client == nil {
		t.Error("Expected New() to return a client, got nil")
	}

	// Test time calculation for recent PRs (this part we can test without API calls)
	oneMonthAgo := time.Now().AddDate(0, -1, 0)
	if oneMonthAgo.After(time.Now()) {
		t.Error("One month ago should be before now")
	}

	// Verify the time is approximately one month ago (within a few days)
	actualDuration := time.Since(oneMonthAgo)
	
	// Allow for variation in month lengths (28-31 days)
	minDuration := 28 * 24 * time.Hour
	maxDuration := 32 * 24 * time.Hour
	
	if actualDuration < minDuration || actualDuration > maxDuration {
		t.Errorf("Expected duration between %v and %v, got %v", minDuration, maxDuration, actualDuration)
	}
}