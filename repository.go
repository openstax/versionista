package main

import (
	"fmt"
	"regexp"
	"strings"
	"strconv"
	"context"
	"github.com/Masterminds/semver"
	"github.com/google/go-github/v25/github"
)


type Repository struct {
	client *github.Client
	owner string
	name string
}

func NewRepository(path string, client *github.Client) *Repository {
	ownerRepo := strings.Split(path, "/")
	return &Repository{
		client: client,
		owner: ownerRepo[0],
		name: ownerRepo[1],
	};
}

func (r *Repository) latestRelease() *semver.Version {
	ctx := context.Background()

	release, _, err :=  r.client.Repositories.GetLatestRelease(
		ctx, r.owner, r.name,
	)
	CheckError(err)

	version, err :=  semver.NewVersion(*release.TagName)
	CheckError(err)

	return version
}

func (r *Repository) String() string {
	return fmt.Sprintf("%s/%s", r.owner, r.name)
}

type ChangeLogEntry struct {
	Number int
	Message string
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

func (repo *Repository) getChangelog(previousRelease *semver.Version) []ChangeLogEntry {
	ctx := context.Background()

	cmp, _, err := repo.client.Repositories.CompareCommits(
		ctx, repo.owner, repo.name,
		fmt.Sprintf("v%s", previousRelease.String()),
		"master",
	)
	CheckError(err)

	prNumR := regexp.MustCompile(`Merge pull request #(\d+) from (?:\S+)(?:\s+)(.*)`)

	var log []ChangeLogEntry

	for _, c := range cmp.Commits {
		msg := *c.GetCommit().Message
		prMatch := prNumR.FindStringSubmatch(msg)
		if len(prMatch) > 0 {
			num, err := strconv.Atoi(prMatch[1])
			CheckError(err)
			log = append(log, ChangeLogEntry{
				Number: num, Message: prMatch[2],
			})
		}
	}
	return log
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
		})
	CheckError(err)
}
