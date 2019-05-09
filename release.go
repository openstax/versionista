package main

import (
	"fmt"
	"github.com/Masterminds/semver"
)


type Release struct {
	repository *Repository
	version *semver.Version
}


func cutRelease(repo *Repository) *Release {
	release := &Release{
		repository: repo,
		version: repo.latestRelease(),
	}

	changeLog := repo.getChangelog(release.version)
	if 0 == len(changeLog) {
		fmt.Printf("  skipping, no PRs found since %s\n", release.version.String())
	} else {
		newVersion := getVersion(release.version, changeLog)
		if newVersion != nil {
			msg := composeReleaseMessage(changeLog)
			repo.createRelease(newVersion, msg)
			release.version = newVersion
			announceRelease(repo, repo.latestRelease());
		}
	}
	return release
}
