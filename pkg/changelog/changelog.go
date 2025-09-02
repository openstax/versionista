package changelog

import (
	"fmt"
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
		pattern := fmt.Sprintf("\\b(%s)-\\d+\\b", strings.Join(jiraBoards, "|"))
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
		`#(\d+)`,
		`pull request #(\d+)`,
		`\(#(\d+)\)`,
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

func BuildReleaseNotes(entries []Entry, crossLinks []CrossLink) string {
	var builder strings.Builder

	if len(crossLinks) > 0 {
		builder.WriteString("## Related Releases\n\n")
		for _, link := range crossLinks {
			builder.WriteString(fmt.Sprintf("- [%s v%s](%s)\n", link.Name, link.Version, link.URL))
		}
		builder.WriteString("\n---\n\n")
	}

	if len(entries) > 0 {
		builder.WriteString("| PR # | Author | Title | Merged Date | Ticket # |\n")
		builder.WriteString("|------|--------|-------|-------------|----------|\n")

		for _, entry := range entries {
			ticketNumbers := ""
			if len(entry.Tickets) > 0 {
				ticketNumbers = strings.Join(entry.Tickets, ", ")
			}
			
			// Format title with details/summary tags if description exists
			escapedTitle := escapeMarkdownTable(entry.Title)
			titleCell := escapedTitle
			if entry.Description != "" {
				escapedDescription := escapeMarkdownTable(entry.Description)
				titleCell = fmt.Sprintf("<details><summary>%s</summary><br>%s</details>", escapedTitle, escapedDescription)
			}
			
			builder.WriteString(fmt.Sprintf("| #%d | %s | %s | %s | %s |\n",
				entry.Number,
				escapeMarkdownTable(entry.Author),
				titleCell,
				entry.Date,
				ticketNumbers))
		}
		builder.WriteString("\n")
	}

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
