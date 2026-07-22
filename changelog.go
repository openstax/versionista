package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// MaxReleaseBodySize is GitHub's hard limit on the length of a release body.
// Requests exceeding it are rejected with "body is too long (maximum is
// 125000 characters)".
const MaxReleaseBodySize = 125000

// truncationNotice is appended to a description that had to be shortened.
const truncationNotice = "… [truncated]"

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

// FitEntriesToBodyLimit builds the entries table so that prefix + table stays
// within maxBodySize. If the full table would exceed the limit, the longest PR
// descriptions are truncated (longest first) until the whole body fits. The
// prefix argument accounts for any content prepended to the table (e.g. cross
// links) so the combined body respects the limit.
//
// The returned entries slice is a copy with descriptions truncated as needed;
// the caller's entries are not mutated.
func FitEntriesToBodyLimit(prefix string, entries []Entry, jiraEnabled bool, jiraOrgId string, maxBodySize int) string {
	// Work on a copy so we never mutate the caller's entries.
	trimmed := make([]Entry, len(entries))
	copy(trimmed, entries)

	table := BuildEntriesTableString(trimmed, jiraEnabled, jiraOrgId)
	if len(prefix)+len(table) <= maxBodySize {
		return table
	}

	// Repeatedly shave the entry with the longest description until the body
	// fits. Truncating the longest first spreads the loss across the biggest
	// contributors and keeps the table structure intact for every PR.
	for len(prefix)+len(table) > maxBodySize {
		idx := longestDescriptionIndex(trimmed)
		if idx < 0 {
			// No description left to trim; nothing more we can do at this
			// level. Return the smallest table we could produce.
			break
		}

		overBy := (len(prefix) + len(table)) - maxBodySize
		desc := trimmed[idx].Description
		// Shrink this description by at least the overage, but never below
		// zero. Removing `overBy` characters of the raw description removes at
		// least that many characters from the rendered body.
		newLen := len(desc) - overBy
		if newLen < 0 {
			newLen = 0
		}
		trimmed[idx].Description = truncateDescription(desc, newLen)

		table = BuildEntriesTableString(trimmed, jiraEnabled, jiraOrgId)
	}

	return table
}

// longestDescriptionIndex returns the index of the entry with the longest
// non-empty description, or -1 if no entry has a truncatable description.
func longestDescriptionIndex(entries []Entry) int {
	idx := -1
	max := 0
	for i, e := range entries {
		// A description already at or below the truncation notice cannot be
		// shortened any further usefully.
		if len(e.Description) > len(truncationNotice) && len(e.Description) > max {
			max = len(e.Description)
			idx = i
		}
	}
	return idx
}

// truncateDescription shortens desc so its rendered length is at most roughly
// targetLen, appending a notice so readers know content was dropped. A
// targetLen of 0 (or one too small to fit the notice) collapses to just the
// notice.
func truncateDescription(desc string, targetLen int) string {
	if targetLen <= len(truncationNotice) {
		return truncationNotice
	}
	keep := targetLen - len(truncationNotice)
	if keep >= len(desc) {
		return desc
	}
	// Trim on a rune boundary so we never split a multi-byte character.
	runes := []rune(desc)
	if keep > len(runes) {
		keep = len(runes)
	}
	return strings.TrimRight(string(runes[:keep]), " ") + truncationNotice
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

