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
	// Create hotfix version by adding suffix as metadata
	// Remove existing metadata if present before adding new suffix
	// For example: v1.2.3+oldfix -> v1.2.3+newfix (not v1.2.3+oldfix+newfix)

	// Build base version string without metadata
	versionStr := fmt.Sprintf("%d.%d.%d", baseVersion.Major(), baseVersion.Minor(), baseVersion.Patch())

	// Include prerelease if present
	if baseVersion.Prerelease() != "" {
		versionStr = fmt.Sprintf("%s-%s", versionStr, baseVersion.Prerelease())
	}

	// Add new metadata suffix
	hotfixStr := fmt.Sprintf("%s+%s", versionStr, suffix)

	hotfixVersion, err := semver.NewVersion(hotfixStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create hotfix version %s: %w", hotfixStr, err)
	}

	return hotfixVersion, nil
}

