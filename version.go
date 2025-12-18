package main

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver"
)

func ParseVersion(versionStr string) (*semver.Version, error) {
	versionStr = strings.TrimPrefix(versionStr, "v")
	
	version, err := semver.NewVersion(versionStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version %s: %w", versionStr, err)
	}
	return version, nil
}

type BumpType string

const (
	BumpPatch BumpType = "patch"
	BumpMinor BumpType = "minor"
	BumpMajor BumpType = "major"
	BumpHotfix BumpType = "hotfix"
)

func BumpVersion(current *semver.Version, bumpType BumpType) *semver.Version {
	// Special case: if current version is 0.0.0 (no previous release), default to 1.0.0
	if current.String() == "0.0.0" {
		newVer, _ := semver.NewVersion("1.0.0")
		return newVer
	}
	
	switch bumpType {
	case BumpMajor:
		newVer := current.IncMajor()
		return &newVer
	case BumpMinor:
		newVer := current.IncMinor()
		return &newVer
	case BumpPatch:
		newVer := current.IncPatch()
		return &newVer
	default:
		newVer := current.IncPatch()
		return &newVer
	}
}

func FormatVersion(v *semver.Version) string {
	if v == nil {
		return "(not set)"
	}
	return fmt.Sprintf("v%s", v.String())
}

func CreateHotfixVersion(baseVersion *semver.Version, suffix string) (*semver.Version, error) {
	// Create hotfix version by adding suffix as build metadata
	// For example: v1.2.3 + "fix1" -> v1.2.3+fix1
	// Strip any existing metadata to avoid appending (e.g., 1.3.0+a -> 1.3.0+b, not 1.3.0+a+b)
	coreVersion := fmt.Sprintf("%d.%d.%d", baseVersion.Major(), baseVersion.Minor(), baseVersion.Patch())
	if baseVersion.Prerelease() != "" {
		coreVersion = fmt.Sprintf("%s-%s", coreVersion, baseVersion.Prerelease())
	}
	hotfixStr := fmt.Sprintf("%s+%s", coreVersion, suffix)

	hotfixVersion, err := semver.NewVersion(hotfixStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create hotfix version %s: %w", hotfixStr, err)
	}

	return hotfixVersion, nil
}

