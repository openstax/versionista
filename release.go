package main

import (
	"context"
	"fmt"
	"github.com/Masterminds/semver"
)

type Release struct {
	repository *Repository
	version    *semver.Version
}

func newRelease(repo *Repository, allRepos []*Repository) *Release {
	repo.fetchChangeLog(context.Background(), fmt.Sprintf("v%s", repo.latestRelease.String()), repo.commitSHA)
	release := &Release{
		repository: repo,
		version:    repo.latestRelease,
	}

	if 0 == len(repo.changeLog) {
		fmt.Printf("  skipping, no PRs found since %s\n", release.version.String())
	} else {
		newVersion := getVersion(release.version, repo.changeLog)
		if newVersion != nil {
			msg := composeReleaseMessage(repo.changeLog, repo, allRepos, newVersion)
			fmt.Printf("  creating release %s\n%s", newVersion.String(), msg)
			repo.createRelease(newVersion, msg)
			release.version = newVersion
			announceRelease(repo, repo.latestRelease)
		}
	}
	return release
}

func newPreReleaseFix(repo *Repository, allRepos []*Repository) *Release {
	release := &Release{
		repository: repo,
		version:    repo.latestStableRelease,
	}

	newVersion, sha := getPreReleaseFixInfo(repo, release.version)

	if newVersion != nil && sha != "" {
		repo.fetchChangeLog(context.Background(), fmt.Sprintf("v%s", release.version.String()), sha)
		msg := composeReleaseMessage(repo.changeLog, repo, allRepos, newVersion)
		repo.createPreRelease(newVersion, sha, msg)
		release.version = newVersion
		announceRelease(repo, newVersion)
	}
	return release
}

func newPostReleaseFix(repo *Repository, allRepos []*Repository) *Release {


	release := &Release{
		repository: repo,
		version:    repo.latestStableRelease,
	}

	newVersion, sha := getPostReleaseFixInfo(repo, release.version)

	if newVersion != nil && sha != "" {
		repo.fetchChangeLog(context.Background(), fmt.Sprintf("v%s", release.version.String()), sha)
		msg := composeReleaseMessage(repo.changeLog, repo, allRepos, newVersion)
		repo.createPostRelease(newVersion, sha, msg)
		release.version = newVersion
		announceRelease(repo, newVersion)
	}
	return release
}
