package main

import (
	"strings"
	"testing"
)

func TestNewGenerator(t *testing.T) {
	tests := []struct {
		name       string
		jiraBoards []string
		hasRegex   bool
	}{
		{
			name:       "with JIRA boards",
			jiraBoards: []string{"TEST", "PROJ"},
			hasRegex:   true,
		},
		{
			name:       "without JIRA boards",
			jiraBoards: []string{},
			hasRegex:   false,
		},
		{
			name:       "nil JIRA boards",
			jiraBoards: nil,
			hasRegex:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewGenerator(tt.jiraBoards)
			
			hasRegex := generator.ticketMatcher != nil
			if hasRegex != tt.hasRegex {
				t.Errorf("Expected hasRegex=%v, got: %v", tt.hasRegex, hasRegex)
			}
		})
	}
}

func TestExtractTickets(t *testing.T) {
	generator := NewGenerator([]string{"TEST", "PROJ"})

	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "single ticket",
			text:     "This fixes TEST-123",
			expected: []string{"TEST-123"},
		},
		{
			name:     "multiple tickets",
			text:     "This fixes TEST-123 and PROJ-456",
			expected: []string{"TEST-123", "PROJ-456"},
		},
		{
			name:     "no tickets",
			text:     "No tickets mentioned here",
			expected: []string{},
		},
		{
			name:     "duplicate tickets",
			text:     "TEST-123 and TEST-123 again",
			expected: []string{"TEST-123"},
		},
		{
			name:     "wrong board prefix",
			text:     "This fixes WRONG-123",
			expected: []string{},
		},
		{
			name:     "ticket in URL",
			text:     "Fixed, see https://openstax.atlassian.net/browse/TEST-291",
			expected: []string{"TEST-291"},
		},
		{
			name:     "ticket in JIRA URL",
			text:     "Fixed, see https://company.atlassian.net/browse/PROJ-456",
			expected: []string{"PROJ-456"},
		},
		{
			name:     "case insensitive - lowercase ticket with uppercase board",
			text:     "This fixes test-123",
			expected: []string{"test-123"},
		},
		{
			name:     "case insensitive - uppercase ticket with lowercase board", 
			text:     "This fixes PROJ-789",
			expected: []string{"PROJ-789"},
		},
		{
			name:     "case insensitive - mixed case ticket",
			text:     "Fixed TeSt-456 and PrOj-123",
			expected: []string{"TeSt-456", "PrOj-123"},
		},
		{
			name:     "ticket with space separator",
			text:     "This fixes TEST 123",
			expected: []string{"TEST 123"},
		},
		{
			name:     "multiple tickets with mixed separators",
			text:     "This fixes TEST-123 and PROJ 456",
			expected: []string{"TEST-123", "PROJ 456"},
		},
		{
			name:     "case insensitive - lowercase ticket with space",
			text:     "This fixes test 123",
			expected: []string{"test 123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tickets := generator.ExtractTickets(tt.text)
			
			if len(tickets) != len(tt.expected) {
				t.Errorf("Expected %d tickets, got: %d", len(tt.expected), len(tickets))
			}
			
			for i, expected := range tt.expected {
				if i >= len(tickets) || tickets[i] != expected {
					t.Errorf("Expected ticket %s at index %d, got: %v", expected, i, tickets)
				}
			}
		})
	}
}

func TestExtractTicketsNoBoards(t *testing.T) {
	generator := NewGenerator([]string{})

	tickets := generator.ExtractTickets("This fixes TEST-123")
	if len(tickets) != 0 {
		t.Errorf("Expected no tickets when no boards configured, got: %v", tickets)
	}
}

func TestExtractTicketsOtterBoard(t *testing.T) {
	// Test with OTTER board specifically
	generator := NewGenerator([]string{"OTTER"})

	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "OTTER ticket in URL",
			text:     "Fixed, see https://openstax.atlassian.net/browse/OTTER-291",
			expected: []string{"OTTER-291"},
		},
		{
			name:     "OTTER ticket plain text",
			text:     "This fixes OTTER-291",
			expected: []string{"OTTER-291"},
		},
		{
			name:     "OTTER ticket with OTTER board configured",
			text:     "Fixed, see https://openstax.atlassian.net/browse/OTTER-291",
			expected: []string{"OTTER-291"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tickets := generator.ExtractTickets(tt.text)
			
			if len(tickets) != len(tt.expected) {
				t.Errorf("Expected %d tickets, got: %d", len(tt.expected), len(tickets))
			}
			
			for i, expected := range tt.expected {
				if i >= len(tickets) || tickets[i] != expected {
					t.Errorf("Expected ticket %s at index %d, got: %v", expected, i, tickets)
				}
			}
		})
	}
}

func TestExtractTicketsOtterNotConfigured(t *testing.T) {
	// Test with only TEST and PROJ boards - OTTER not configured
	generator := NewGenerator([]string{"TEST", "PROJ"})

	tickets := generator.ExtractTickets("Fixed, see https://openstax.atlassian.net/browse/OTTER-291")
	if len(tickets) != 0 {
		t.Errorf("Expected no tickets when OTTER board not configured, got: %v", tickets)
	}
}

func TestExtractTicketsCaseInsensitive(t *testing.T) {
	// Test case insensitive matching with specific examples
	tests := []struct {
		name       string
		jiraBoards []string
		text       string
		expected   []string
	}{
		{
			name:       "otter-123 matches board OTTER",
			jiraBoards: []string{"OTTER"},
			text:       "Fixed otter-123",
			expected:   []string{"otter-123"},
		},
		{
			name:       "board foo matches ticket FOO-291 in URL",
			jiraBoards: []string{"foo"},
			text:       "Fixed, see https://openstax.atlassian.net/browse/FOO-291",
			expected:   []string{"FOO-291"},
		},
		{
			name:       "mixed case boards and tickets",
			jiraBoards: []string{"TeSt", "PrOj"},
			text:       "Fixed test-123 and PROJ-456 and TeSt-789",
			expected:   []string{"test-123", "PROJ-456", "TeSt-789"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewGenerator(tt.jiraBoards)
			tickets := generator.ExtractTickets(tt.text)
			
			if len(tickets) != len(tt.expected) {
				t.Errorf("Expected %d tickets, got: %d", len(tt.expected), len(tickets))
			}
			
			for i, expected := range tt.expected {
				if i >= len(tickets) || tickets[i] != expected {
					t.Errorf("Expected ticket %s at index %d, got: %v", expected, i, tickets)
				}
			}
		})
	}
}

func TestParsePRNumber(t *testing.T) {
	tests := []struct {
		name          string
		commitMessage string
		expected      int
		expectError   bool
	}{
		{
			name:          "merge pull request",
			commitMessage: "Merge pull request #123 from branch",
			expected:      123,
			expectError:   false,
		},
		{
			name:          "PR number in parentheses",
			commitMessage: "Fix bug (#456)",
			expected:      456,
			expectError:   false,
		},
		{
			name:          "simple hash reference",
			commitMessage: "Fix issue #789",
			expected:      789,
			expectError:   false,
		},
		{
			name:          "no PR number",
			commitMessage: "Regular commit message",
			expected:      0,
			expectError:   true,
		},
		{
			name:          "no PR number with hash in word",
			commitMessage: "color:#123456",
			expected:      0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prNumber, err := ParsePRNumber(tt.commitMessage)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}
			
			if prNumber != tt.expected {
				t.Errorf("Expected PR number %d, got: %d", tt.expected, prNumber)
			}
		})
	}
}

func TestBuildCrossLinksString(t *testing.T) {
	crossLinks := []CrossLink{
		{
			Name:    "Related Repo",
			Version: "1.2.3",
			URL:     "https://github.com/org/repo/releases/tag/v1.2.3",
		},
	}

	result := BuildCrossLinksString(crossLinks)

	if !strings.Contains(result, "## Related Releases") {
		t.Error("Expected cross-links section in release notes")
	}
	if !strings.Contains(result, "Related Repo v1.2.3") {
		t.Error("Expected cross-link to Related Repo")
	}
}

func TestBuildCrossLinksStringEmpty(t *testing.T) {
	result := BuildCrossLinksString([]CrossLink{})
	if result != "" {
		t.Error("Expected empty string for no cross-links")
	}
}

func TestBuildEntriesTableString(t *testing.T) {
	entries := []Entry{
		{
			Number:      123,
			Date:        "2023-01-01",
			Author:      "testuser",
			Title:       "Fix important bug",
			Description: "This fixes a critical issue",
			Tickets:     []string{"test-456"},
		},
		{
			Number:      124,
			Date:        "2023-01-02",
			Author:      "anotheruser",
			Title:       "Add new feature",
			Description: "",
			Tickets:     []string{},
		},
	}

	result := BuildEntriesTableString(entries, true, "my-org")

	if !strings.Contains(result, "| PR # | Author | Title | Merged Date | Ticket # |") {
		t.Error("Expected table header in release notes")
	}
	// Ticket should be normalized to uppercase
	if !strings.Contains(result, "| #123 | testuser | <details><summary>Fix important bug</summary><br>This fixes a critical issue</details> | 2023-01-01 | [TEST-456](https://my-org.atlassian.net/browse/TEST-456) |") {
		t.Error("Expected PR #123 in table format with details/summary tags and normalized ticket")
	}
	if !strings.Contains(result, "| #124 | anotheruser | Add new feature | 2023-01-02 |  |") {
		t.Error("Expected PR #124 in table format (no description, so no details tags)")
	}
	if !strings.Contains(result, "[TEST-456](https://my-org.atlassian.net/browse/TEST-456)") {
		t.Error("Expected normalized ticket reference")
	}
}

func TestBuildEntriesTableStringJiraDisabled(t *testing.T) {
	entries := []Entry{
		{
			Number:      123,
			Date:        "2023-01-01",
			Author:      "testuser",
			Title:       "Fix important bug",
			Description: "This fixes a critical issue",
			Tickets:     []string{"TEST-456"},
		},
	}

	result := BuildEntriesTableString(entries, false, "")

	if strings.Contains(result, "| PR # | Author | Title | Merged Date | Ticket # |") {
		t.Error("Did not expect ticket column in table header")
	}
	if strings.Contains(result, "TEST-456") {
		t.Error("Did not expect ticket reference in table")
	}
	if !strings.Contains(result, "| #123 | testuser | <details><summary>Fix important bug</summary><br>This fixes a critical issue</details> | 2023-01-01 |") {
		t.Error("Expected PR #123 in table format without ticket number")
	}
}

func TestEscapeMarkdownTable(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "pipe character",
			input:    "Title with | pipe",
			expected: "Title with \\| pipe",
		},
		{
			name:     "newlines",
			input:    "Title with\nnewline\r\nand carriage return",
			expected: "Title with<br>newline<br>and carriage return",
		},
		{
			name:     "multiple spaces",
			input:    "Title   with    multiple    spaces",
			expected: "Title with multiple spaces",
		},
		{
			name:     "whitespace trimming",
			input:    "  Title with leading and trailing spaces  ",
			expected: "Title with leading and trailing spaces",
		},
		{
			name:     "complex case",
			input:    " Title with | pipe\nand newline   and spaces  ",
			expected: "Title with \\| pipe<br>and newline and spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeMarkdownTable(tt.input)
			if result != tt.expected {
				t.Errorf("escapeMarkdownTable(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRemoveDuplicates(t *testing.T) {
	input := []string{"TEST-123", "test-123", "PROJ-456", "TEST-123"}
	expected := []string{"TEST-123", "PROJ-456"}

	result := removeDuplicates(input)

	if len(result) != len(expected) {
		t.Errorf("Expected %d unique items, got: %d", len(expected), len(result))
	}

	for i, expectedItem := range expected {
		if i >= len(result) || result[i] != expectedItem {
			t.Errorf("Expected item %s at index %d, got: %v", expectedItem, i, result)
		}
	}
}

func TestNormalizeTicket(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "space separator to hyphen",
			input:    "Otter 35",
			expected: "OTTER-35",
		},
		{
			name:     "lowercase space separator",
			input:    "otter 35",
			expected: "OTTER-35",
		},
		{
			name:     "already normalized",
			input:    "OTTER-35",
			expected: "OTTER-35",
		},
		{
			name:     "lowercase with hyphen",
			input:    "otter-35",
			expected: "OTTER-35",
		},
		{
			name:     "mixed case with space",
			input:    "OtTeR 35",
			expected: "OTTER-35",
		},
		{
			name:     "mixed case with hyphen",
			input:    "OtTeR-35",
			expected: "OTTER-35",
		},
		{
			name:     "TEST board with space",
			input:    "TEST 123",
			expected: "TEST-123",
		},
		{
			name:     "multiple spaces",
			input:    "OTTER  35",
			expected: "OTTER--35",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeTicket(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeTicket(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildEntriesTableStringWithNormalizedTickets(t *testing.T) {
	entries := []Entry{
		{
			Number:      123,
			Date:        "2023-01-01",
			Author:      "testuser",
			Title:       "Fix bug",
			Description: "",
			Tickets:     []string{"Otter 35", "TEST 456"},
		},
		{
			Number:      124,
			Date:        "2023-01-02",
			Author:      "anotheruser",
			Title:       "Add feature",
			Description: "",
			Tickets:     []string{"otter-789", "test 123"},
		},
	}

	result := BuildEntriesTableString(entries, true, "my-org")

	// Check that tickets are normalized in both the display text and URLs
	if !strings.Contains(result, "[OTTER-35](https://my-org.atlassian.net/browse/OTTER-35)") {
		t.Error("Expected normalized ticket OTTER-35 in link for 'Otter 35'")
	}
	if !strings.Contains(result, "[TEST-456](https://my-org.atlassian.net/browse/TEST-456)") {
		t.Error("Expected normalized ticket TEST-456 in link for 'TEST 456'")
	}
	if !strings.Contains(result, "[OTTER-789](https://my-org.atlassian.net/browse/OTTER-789)") {
		t.Error("Expected normalized ticket OTTER-789 in link for 'otter-789'")
	}
	if !strings.Contains(result, "[TEST-123](https://my-org.atlassian.net/browse/TEST-123)") {
		t.Error("Expected normalized ticket TEST-123 in link for 'test 123'")
	}
}

