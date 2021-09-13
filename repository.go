package main

import (
	"fmt"
	"regexp"
	"strings"
	"strconv"
	"context"
	"github.com/spf13/viper"
	"github.com/Masterminds/semver"
	"github.com/google/go-github/v28/github"
)



type ChangeLogEntry struct {
	Number int
	Message string
}

type Repository struct {
	client *github.Client
	owner string
	name string
	branch string
	latestRelease *semver.Version
	changeLog []ChangeLogEntry
}

func NewRepository(path string, client *github.Client) *Repository {
	ownerRepo := strings.Split(path, "/")
	var branch string
	branch = viper.GetString(fmt.Sprintf("branches.%s", path))
	if branch == "" {
		branch = "main"
	}
	return &Repository{
		client: client,
		owner: ownerRepo[0],
		name: ownerRepo[1],
		branch: branch,
	};
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

var squashLine = regexp.MustCompile(`\s*(.*)\s+\(\#(\d+)\)`)
var mergeLine = regexp.MustCompile(`Merge pull request #(\d+) from (?:\S+)(?:\s+)(.*)`)

func (r *Repository) fetch() {
	ctx := context.Background()

	release, _, err :=  r.client.Repositories.GetLatestRelease(
		ctx, r.owner, r.name,
	)
	CheckError(err)

	version, err :=  semver.NewVersion(*release.TagName)
	if err != nil {
		fmt.Printf("Failed to parse %s version \"%s\"\n", r.name, *release.TagName)
		CheckError(err)
	}

	r.latestRelease = version

	cmp, _, err := r.client.Repositories.CompareCommits(
		ctx, r.owner, r.name,
		fmt.Sprintf("v%s", version),
		r.branch,
	)
	CheckError(err)

	for _, c := range cmp.Commits {
		msg := *c.GetCommit().Message
		squashMatch := squashLine.FindStringSubmatch(msg)
		if len(squashMatch) > 0 {
			num, err := strconv.Atoi(squashMatch[2])
			CheckError(err)
			r.changeLog = append(r.changeLog, ChangeLogEntry{
				Number: num, Message: squashMatch[1],
			})
		} else {
			mergeMatch := mergeLine.FindStringSubmatch(msg)
			if len(mergeMatch) > 0 {
				num, err := strconv.Atoi(mergeMatch[1])
				CheckError(err)
				r.changeLog = append(r.changeLog, ChangeLogEntry{
					Number: num, Message: mergeMatch[2],
				})
			}
		}
	}
}

func (repo *Repository) createRelease(version *semver.Version, message string ) {
	ctx := context.Background()
	tag := fmt.Sprintf("v%s", version.String())
	_, _, err := repo.client.Repositories.CreateRelease(
		ctx, repo.owner, repo.name,
		&github.RepositoryRelease{
			Name: &tag,
			Body: &message,
			TagName: &tag,
			TargetCommitish: &repo.branch,
		})
	CheckError(err)
}
