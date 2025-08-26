package main

import (
	"context"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/google/go-github/v28/github"
	"github.com/spf13/viper"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type RepoConfig struct {
	Repo      string `mapstructure:"repo"`
	Alias     string `mapstructure:"alias"`
	Jira      bool   `mapstructure:"jira"`
	CrossLink bool   `mapstructure:"crossLink"`
}

type ChangeLogEntry struct {
	Number      int
	Date        string
	Author      string
	Title       string
	Description string
	Tickets     []string
}

type Repository struct {
	client              *github.Client
	owner               string
	name                string
	alias               string
	commitSHA           string
	ticketMatcher       *regexp.Regexp
	jiraEnabled         bool
	crossLinkEnabled    bool
	latestRelease       *semver.Version
	latestStableRelease *semver.Version
	changeLog           []ChangeLogEntry
}

func NewRepository(config RepoConfig, client *github.Client) *Repository {
	ownerRepo := strings.Split(config.Repo, "/")
	var branch string
	branch = viper.GetString(fmt.Sprintf("branches.%s", config.Repo))
	if branch == "" {
		branch = "main"
	}
	repo := &Repository{
		client:           client,
		owner:            ownerRepo[0],
		name:             ownerRepo[1],
		alias:            config.Alias,
		commitSHA:        branch,
		jiraEnabled:      config.Jira,
		crossLinkEnabled: config.CrossLink,
	}
	
	jiraBoards := viper.GetStringSlice("jira_boards")
	if len(jiraBoards) > 0 && repo.jiraEnabled {
		repo.ticketMatcher = regexp.MustCompile(fmt.Sprintf(`(?i)[^a-zA-Z0-9]*(%s)[-\s](\d+)[^a-zA-Z0-9]*`, strings.Join(jiraBoards, "|")))
	}
	return repo
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
	if !r.jiraEnabled || r.ticketMatcher == nil {
		return matches
	}
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
		entry.Title = *pr.Title
	}
	
	if pr.Body != nil {
		entry.Description = *pr.Body
	}
	
	if r.ticketMatcher == nil {
		return
	}

	if entry.Title != "" {
		entry.Tickets = r.appendMatches(entry.Tickets, entry.Title)
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

var squashLine = regexp.MustCompile(`\s*(.*)\s+\(#(\d+)\)`) 

// match <number>, then non-greedily match any non-space, an optional space, then capture anything else
var mergeLine = regexp.MustCompile(`Merge pull request #(\d+) from (?:\S+)(?:\s*)(.*)`) 

func (r *Repository) fetchChangeLog(ctx context.Context, base string, head string) {
	cmp, _, err := r.client.Repositories.CompareCommits(
		ctx, r.owner, r.name,
		base,
		head,
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

func (r *Repository) resolveVersions(ctx context.Context) {
	releases, _, err := r.client.Repositories.ListReleases(ctx, r.owner, r.name, nil)
	CheckError(err)

	if len(releases) == 0 {
		Warn("No releases found for %s/%s. Please create an initial release.", r.owner, r.name)
		os.Exit(1)
	}

	for _, release := range releases {
		if release.TagName == nil {
			continue
		}
		v, err := semver.NewVersion(*release.TagName)
		if err != nil {
			continue
		}

		if r.latestRelease == nil {
			r.latestRelease = v
		}

		if r.latestStableRelease == nil && v.Prerelease() == "" {
			r.latestStableRelease = v
		}

		if r.latestRelease != nil && r.latestStableRelease != nil {
			break
		}
	}

	if r.latestRelease == nil {
		// This would happen if no tags are valid semver.
		Warn("No valid semver release tags found for %s/%s.", r.owner, r.name)
		os.Exit(1)
	}

	if r.latestStableRelease == nil {
		Warn("No stable (non-prerelease) release found for %s/%s. Please create a stable release first.", r.owner, r.name)
		os.Exit(1)
	}
}
func (r *Repository) appendChangeLog(ctx context.Context, tag string, body string) {
	fileContent, _, _, err := r.client.Repositories.GetContents(ctx, r.owner, r.name, "CHANGELOG.md", &github.RepositoryContentGetOptions{
		Ref: r.commitSHA,
	})
	CheckError(err)

	oldContent, err := fileContent.GetContent()
	if err != nil {
		fmt.Println("Error decoding content:", err)
		return
	}

	updatedContent := fmt.Sprintf("## [%s - %s](%s)\n\n%s\n\n---\n\n%s",
		tag,
		time.Now().Format(time.DateOnly),
		fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", r.owner, r.name, tag),
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

func (repo *Repository) createPreRelease(version *semver.Version, sha string, message string) {
	ctx := context.Background()
	tag := fmt.Sprintf("v%s", version.String())

	_, _, err := repo.client.Repositories.CreateRelease(
		ctx, repo.owner, repo.name,
		&github.RepositoryRelease{
			Name:            &tag,
			Body:            &message,
			TagName:         &tag,
			TargetCommitish: &sha,
		})

	if err != nil {
		fmt.Println("Error creating pre-release:", err)
		return
	}
	CheckError(err)
}

func (repo *Repository) createPostRelease(version *semver.Version, sha string, message string) {
	ctx := context.Background()
	tag := fmt.Sprintf("v%s", version.String())
	//fmt.Printf("RELE %s %s\n", sha, tag)
	return
	_, _, err := repo.client.Repositories.CreateRelease(
		ctx, repo.owner, repo.name,
		&github.RepositoryRelease{
			Name:            &tag,
			Body:            &message,
			TagName:         &tag,
			TargetCommitish: &sha,
		})

	if err != nil {
		fmt.Println("Error creating post-release:", err)
		return
	}
	CheckError(err)
}

func (r *Repository) GetLatestSHA(branchName string) string {
	ctx := context.Background()
	branch, _, err := r.client.Repositories.GetBranch(ctx, r.owner, r.name, branchName)
	if err != nil {
		return ""
	}
	return branch.GetCommit().GetSHA()
}
