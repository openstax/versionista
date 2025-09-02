package version

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
	return fmt.Sprintf("v%s", v.String())
}

func CreateHotfixVersion(baseVersion *semver.Version, suffix string) (*semver.Version, error) {
	// Create hotfix version by adding suffix as prerelease
	// For example: v1.2.3 + "fix1" -> v1.2.3+fix1
	hotfixStr := fmt.Sprintf("%s+%s", baseVersion.String(), suffix)
	
	hotfixVersion, err := semver.NewVersion(hotfixStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create hotfix version %s: %w", hotfixStr, err)
	}
	
	return hotfixVersion, nil
}

func IsValidVersion(versionStr string) bool {
	versionStr = strings.TrimPrefix(versionStr, "v")
	_, err := semver.NewVersion(versionStr)
	return err == nil
}
