package main

import (
	"fmt"
	"github.com/Masterminds/semver"
)

type Release struct {
	repository *Repository
	version    *semver.Version
}

func newRelease(repo *Repository) *Release {

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
