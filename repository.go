package main

import (
	"context"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/google/go-github/v28/github"
	"github.com/spf13/viper"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ChangeLogEntry struct {
	Number  int
	Date    string
	Author  string
	Title   string
	Tickets []string
}

type Repository struct {
	client        *github.Client
	owner         string
	name          string
	commitSHA     string
	ticketMatcher *regexp.Regexp
	latestRelease *semver.Version
	changeLog     []ChangeLogEntry
}

func NewRepository(path string, client *github.Client) *Repository {
	ownerRepo := strings.Split(path, "/")
	var branch string
	branch = viper.GetString(fmt.Sprintf("branches.%s", path))
	if branch == "" {
		branch = "main"
	}
	jiraBoards := viper.GetStringSlice("jira_boards")
	return &Repository{
		client:        client,
		ticketMatcher: regexp.MustCompile(fmt.Sprintf(`(?i)\s*(%s)[-\s](\d+)\s*`, strings.Join(jiraBoards, "|"))),
		owner:         ownerRepo[0],
		name:          ownerRepo[1],
		commitSHA:     branch,
	}
}

func (r *Repository) String() string {
	return fmt.Sprintf("%s/%s", r.owner, r.name)
}

func (repo *Repository) getRecentReleases() []*github.RepositoryRelease {
	ctx := context.Background()
	releases, _, err := repo.client.Repositories.ListReleases(ctx, repo.owner, repo.name, nil)
	CheckError(err)
	return releases
}

func (repo *Repository) deleteRelease(release *github.RepositoryRelease) {
	ctx := context.Background()
	_, err := repo.client.Repositories.DeleteRelease(ctx, repo.owner, repo.name, *release.ID)
	CheckError(err)
}

func (r *Repository) appendMatches(matches []string, content string) []string {
	if r.ticketMatcher.MatchString(content) {
		matched := r.ticketMatcher.FindStringSubmatch(content)
		if len(matched) > 2 {
			ticket := fmt.Sprintf("%s-%s", matched[1], matched[2])
			matches = append(matches, ticket)
		}
	}
	return matches
}

func (r *Repository) parsePR(ctx context.Context, entry *ChangeLogEntry) {

	// Fetch the pull request.
	pr, _, err := r.client.PullRequests.Get(ctx, r.owner, r.name, entry.Number)
	CheckError(err)

	if pr.User.Login != nil {
		entry.Author = *pr.User.Login
	}

	if pr.Title != nil && *pr.Title != "" {
		entry.Tickets = r.appendMatches(entry.Tickets, *pr.Title)
		entry.Title = *pr.Title
	}
	if pr.Body != nil {
		entry.Tickets = r.appendMatches(entry.Tickets, *pr.Body)
	}

	issues, _, err := r.client.Issues.ListComments(ctx, r.owner, r.name, entry.Number, nil)
	CheckError(err)
	for _, c := range issues {
		if c.Body != nil {
			entry.Tickets = r.appendMatches(entry.Tickets, *c.Body)
		}
	}

	comments, _, err := r.client.PullRequests.ListComments(ctx, r.owner, r.name, entry.Number, nil)
	CheckError(err)
	for _, rc := range comments {
		if rc.Body != nil {
			entry.Tickets = r.appendMatches(entry.Tickets, *rc.Body)
		}
	}

	commits, _, err := r.client.PullRequests.ListCommits(ctx, r.owner, r.name, entry.Number, nil)
	CheckError(err)
	for _, commit := range commits {
		entry.Tickets = r.appendMatches(entry.Tickets, commit.Commit.GetMessage())
	}

	entry.Tickets = removeDuplicates(entry.Tickets)

	fmt.Printf("✓ #%d, %s\n", entry.Number, entry.Title)

	return

}

var squashLine = regexp.MustCompile(`\s*(.*)\s+\(\#(\d+)\)`)

// match <number>, then non-greedily match any non-space, an optional space, then capture anything else
var mergeLine = regexp.MustCompile(`Merge pull request #(\d+) from (?:\S+)(?:\s*)(.*)`)

func (r *Repository) fetch() {
	ctx := context.Background()

	release, _, err := r.client.Repositories.GetLatestRelease(
		ctx, r.owner, r.name,
	)
	CheckError(err)

	version, err := semver.NewVersion(*release.TagName)
	CheckError(err)

	r.latestRelease = version

	cmp, _, err := r.client.Repositories.CompareCommits(
		ctx, r.owner, r.name,
		fmt.Sprintf("v%s", version),
		r.commitSHA,
	)
	CheckError(err)

	for _, c := range cmp.Commits {

		msg := *c.GetCommit().Message
		entry := ChangeLogEntry{}
		entry.Date = c.GetCommit().GetAuthor().GetDate().Format(time.DateTime)

		squashMatch := squashLine.FindStringSubmatch(msg)
		if len(squashMatch) > 0 {
			num, err := strconv.Atoi(squashMatch[2])
			CheckError(err)
			entry.Number = num
		} else {
			mergeMatch := mergeLine.FindStringSubmatch(msg)
			if len(mergeMatch) > 0 {
				num, err := strconv.Atoi(mergeMatch[1])
				CheckError(err)
				entry.Number = num
			}
		}

		if entry.Number != 0 {
			r.parsePR(ctx, &entry)
			r.changeLog = append(r.changeLog, entry)
		}

	}
}
func (r *Repository) appendChangeLog(ctx context.Context, tag string, body string) {
	fileContent, _, _, err := r.client.Repositories.GetContents(ctx, r.owner, r.name, "CHANGELOG.md", &github.RepositoryContentGetOptions{
		Ref: r.commitSHA,
	})
	CheckError(err)
	//currentTime := time.Now().Format(time.DateOnly)
	oldContent, err := fileContent.GetContent()
	if err != nil {
		fmt.Println("Error decoding content:", err)
		return
	}

	updatedContent := fmt.Sprintf("## [%s - %s](%s)\n\n%s\n\n---\n\n",
		tag,
		time.Now().Format(time.DateOnly),
		fmt.Sprintf("https://github.com/%s/%s/blob/%s", r.owner, r.name, r.commitSHA),
		body,
		oldContent)

	_, _, err = r.client.Repositories.UpdateFile(ctx, r.owner, r.name, "CHANGELOG.md", &github.RepositoryContentFileOptions{
		Content: []byte(updatedContent),
		Message: github.String(fmt.Sprintf("chore(changelog): update for release %s", tag)),
		SHA:     fileContent.SHA,
	})
	CheckError(err)
}

func (repo *Repository) createRelease(version *semver.Version, message string) {
	ctx := context.Background()
	tag := fmt.Sprintf("v%s", version.String())

	repo.appendChangeLog(ctx, tag, message)

	_, _, err := repo.client.Repositories.CreateRelease(
		ctx, repo.owner, repo.name,
		&github.RepositoryRelease{
			Name:            &tag,
			Body:            &message,
			TagName:         &tag,
			TargetCommitish: &repo.commitSHA,
		})

	if err != nil {
		fmt.Println("Error fetching file:", err)
		return
	}
	CheckError(err)
}
