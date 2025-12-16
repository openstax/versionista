package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type Entry struct {
	Number      int
	Date        string
	Author      string
	Title       string
	Description string
	Tickets     []string
}

type Generator struct {
	ticketMatcher *regexp.Regexp
}

func NewGenerator(jiraBoards []string) *Generator {
	var ticketMatcher *regexp.Regexp
	if len(jiraBoards) > 0 {
		pattern := fmt.Sprintf("(?i)\\b(%s)[-\\s]\\d+\\b", strings.Join(jiraBoards, "|"))
		ticketMatcher = regexp.MustCompile(pattern)
	}

	return &Generator{
		ticketMatcher: ticketMatcher,
	}
}

func (g *Generator) ExtractTickets(text string) []string {
	if g.ticketMatcher == nil {
		return []string{}
	}

	matches := g.ticketMatcher.FindAllString(text, -1)
	return removeDuplicates(matches)
}

func ParsePRNumber(commitMessage string) (int, error) {
	patterns := []string{
		`\bpull request #(\d+)\b`,
		`\(#(\d+)\)`,
		`(?:^|\s)#(\d+)\b`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(commitMessage)
		if len(matches) > 1 {
			return strconv.Atoi(matches[1])
		}
	}

	return 0, fmt.Errorf("no PR number found in commit message: %s", commitMessage)
}

func BuildCrossLinksString(crossLinks []CrossLink) string {
	if len(crossLinks) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("## Related Releases\n\n")

	for _, link := range crossLinks {
		builder.WriteString(fmt.Sprintf("- [%s v%s](%s)\n", link.Name, link.Version, link.URL))
	}
	builder.WriteString("\n------\n\n")
	return builder.String()
}

func normalizeTicket(ticket string) string {
	// Convert to uppercase and replace spaces with hyphens
	normalized := strings.ToUpper(ticket)
	normalized = strings.ReplaceAll(normalized, " ", "-")
	return normalized
}

func BuildEntriesTableString(entries []Entry, jiraEnabled bool, jiraOrgId string) string {
	var builder strings.Builder

	header := "| PR # | Author | Title | Merged Date |"
	separator := "|------|--------|-------|-------------|"

	if jiraEnabled {
		header += " Ticket # |"
		separator += "----------|"
	}
	builder.WriteString(header + "\n")
	builder.WriteString(separator + "\n")

	for _, entry := range entries {
		// Format title with details/summary tags if description exists
		escapedTitle := escapeMarkdownTable(entry.Title)
		titleCell := escapedTitle
		if entry.Description != "" {
			escapedDescription := escapeMarkdownTable(entry.Description)
			titleCell = fmt.Sprintf("<details><summary>%s</summary><br>%s</details>", escapedTitle, escapedDescription)
		}

		line := fmt.Sprintf("| #%d | %s | %s | %s |",
			entry.Number,
			escapeMarkdownTable(entry.Author),
			titleCell,
			entry.Date)

		if jiraEnabled {
			var ticketLinks []string
			for _, ticket := range entry.Tickets {
				normalizedTicket := normalizeTicket(ticket)
				url := fmt.Sprintf("https://%s.atlassian.net/browse/%s", jiraOrgId, normalizedTicket)
				ticketLinks = append(ticketLinks, fmt.Sprintf("[%s](%s)", normalizedTicket, url))
			}
			line += fmt.Sprintf(" %s |", strings.Join(ticketLinks, ", "))
		}
		builder.WriteString(line + "\n")
	}
	builder.WriteString("\n")

	return builder.String()
}


type CrossLink struct {
	Name    string
	Version string
	URL     string
}

func escapeMarkdownTable(text string) string {
	// Escape pipe characters that would break table formatting
	text = strings.ReplaceAll(text, "|", "\\|")
	// Replace newlines with spaces to prevent table row breaks
	text = strings.ReplaceAll(text, "\n", "<br>")
	text = strings.ReplaceAll(text, "\r", "")
	// Replace multiple spaces with single space
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	// Trim whitespace
	text = strings.TrimSpace(text)
	return text
}

func removeDuplicates(input []string) []string {
	seen := make(map[string]struct{})
	result := []string{}

	for _, str := range input {
		upper := strings.ToUpper(str)
		if _, ok := seen[upper]; !ok {
			seen[upper] = struct{}{}
			result = append(result, str)
		}
	}

	return result
}

func EditChangelog(entries []Entry, crossLinks []CrossLink, jiraEnabled bool, jiraOrgId string) (string, error) {
	// Create temporary file with current changelog
	tmpfile, err := os.CreateTemp("", "changelog-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write current changelog in final release notes format to temp file
	var builder strings.Builder
	builder.WriteString(BuildCrossLinksString(crossLinks))
	if len(entries) > 0 {
		builder.WriteString(BuildEntriesTableString(entries, jiraEnabled, jiraOrgId))
	}
	changelogText := builder.String()
	if _, err := tmpfile.Write([]byte(changelogText)); err != nil {
		return "", fmt.Errorf("failed to write to temp file: %w", err)
	}
	if err := tmpfile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // fallback to vi
	}

	// Open editor
	cmd := exec.Command(editor, tmpfile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run editor: %w", err)
	}

	// Read the edited content and return as-is
	editedContent, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read edited file: %w", err)
	}

	return string(editedContent), nil
}
