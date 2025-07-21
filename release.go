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

func newRelease(repo *Repository) *Release {
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
			msg := composeReleaseMessage(repo.changeLog)
			fmt.Printf("  creating release %s\n%s", newVersion.String(), msg)
			repo.createRelease(newVersion, msg)
			release.version = newVersion
			announceRelease(repo, repo.latestRelease)
		}
	}
	return release
}

func newPreReleaseFix(repo *Repository) *Release {
	release := &Release{
		repository: repo,
		version:    repo.latestStableRelease,
	}

	newVersion, sha := getPreReleaseFixInfo(release.version)
	if newVersion != nil && sha != "" {
		repo.fetchChangeLog(context.Background(), fmt.Sprintf("v%s", release.version.String()), sha)
		msg := composeReleaseMessage(repo.changeLog)
		repo.createPreReleaseFix(newVersion, sha, msg)
		release.version = newVersion
		announceRelease(repo, newVersion)
	}
	return release
}

func newPostReleaseFix(repo *Repository) *Release {
	release := &Release{
		repository: repo,
		version:    repo.latestStableRelease,
	}

	newVersion, sha := getPostReleaseFixInfo(release.version)
	if newVersion != nil && sha != "" {
		repo.fetchChangeLog(context.Background(), fmt.Sprintf("v%s", release.version.String()), sha)
		msg := composeReleaseMessage(repo.changeLog)
		repo.createPostReleaseFix(newVersion, sha, msg)
		release.version = newVersion
		announceRelease(repo, newVersion)
	}
	return release
}
