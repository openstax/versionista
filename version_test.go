package main

import (
	"testing"

	"github.com/Masterminds/semver"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name        string
		versionStr  string
		expected    string
		expectError bool
	}{
		{
			name:        "version with v prefix",
			versionStr:  "v1.2.3",
			expected:    "1.2.3",
			expectError: false,
		},
		{
			name:        "version without v prefix",
			versionStr:  "1.2.3",
			expected:    "1.2.3",
			expectError: false,
		},
		{
			name:        "prerelease version",
			versionStr:  "v1.2.3-beta.1",
			expected:    "1.2.3-beta.1",
			expectError: false,
		},
		{
			name:        "invalid version",
			versionStr:  "not.a.version",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := ParseVersion(tt.versionStr)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}
			
			if version.String() != tt.expected {
				t.Errorf("Expected version %s, got: %s", tt.expected, version.String())
			}
		})
	}
}

func TestBumpVersion(t *testing.T) {
	baseVersion, _ := semver.NewVersion("1.2.3")

	tests := []struct {
		name     string
		bumpType BumpType
		expected string
	}{
		{
			name:     "patch bump",
			bumpType: BumpPatch,
			expected: "1.2.4",
		},
		{
			name:     "minor bump",
			bumpType: BumpMinor,
			expected: "1.3.0",
		},
		{
			name:     "major bump",
			bumpType: BumpMajor,
			expected: "2.0.0",
		},
		{
			name:     "invalid bump defaults to patch",
			bumpType: BumpType("invalid"),
			expected: "1.2.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newVersion := BumpVersion(baseVersion, tt.bumpType)
			
			if newVersion.String() != tt.expected {
				t.Errorf("Expected version %s, got: %s", tt.expected, newVersion.String())
			}
		})
	}
}

func TestBumpVersionFromZero(t *testing.T) {
	// Test special case: bumping from 0.0.0 should always result in 1.0.0
	baseVersion, _ := semver.NewVersion("0.0.0")

	tests := []struct {
		name     string
		bumpType BumpType
		expected string
	}{
		{
			name:     "patch bump from 0.0.0",
			bumpType: BumpPatch,
			expected: "1.0.0",
		},
		{
			name:     "minor bump from 0.0.0",
			bumpType: BumpMinor,
			expected: "1.0.0",
		},
		{
			name:     "major bump from 0.0.0",
			bumpType: BumpMajor,
			expected: "1.0.0",
		},
		{
			name:     "invalid bump from 0.0.0 defaults to 1.0.0",
			bumpType: BumpType("invalid"),
			expected: "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newVersion := BumpVersion(baseVersion, tt.bumpType)
			
			if newVersion.String() != tt.expected {
				t.Errorf("Expected version %s, got: %s", tt.expected, newVersion.String())
			}
		})
	}
}

func TestFormatVersion(t *testing.T) {
	version, _ := semver.NewVersion("1.2.3")
	
	formatted := FormatVersion(version)
	expected := "v1.2.3"
	
	if formatted != expected {
		t.Errorf("Expected %s, got: %s", expected, formatted)
	}
}

func TestCreateHotfixVersion(t *testing.T) {
	tests := []struct {
		name        string
		baseVersion string
		suffix      string
		expected    string
		expectError bool
	}{
		{
			name:        "valid hotfix version",
			baseVersion: "1.2.3",
			suffix:      "fix1",
			expected:    "1.2.3+fix1",
			expectError: false,
		},
		{
			name:        "hotfix with longer suffix",
			baseVersion: "1.0.0",
			suffix:      "critical-security-fix",
			expected:    "1.0.0+critical-security-fix",
			expectError: false,
		},
		{
			name:        "numeric suffix",
			baseVersion: "2.1.0",
			suffix:      "123",
			expected:    "2.1.0+123",
			expectError: false,
		},
		{
			name:        "replace existing metadata",
			baseVersion: "1.2.3+oldfix",
			suffix:      "newfix",
			expected:    "1.2.3+newfix",
			expectError: false,
		},
		{
			name:        "replace metadata with different suffix",
			baseVersion: "2.0.0+build123",
			suffix:      "hotfix456",
			expected:    "2.0.0+hotfix456",
			expectError: false,
		},
		{
			name:        "version with prerelease and no metadata",
			baseVersion: "1.2.3-rc1",
			suffix:      "fix1",
			expected:    "1.2.3-rc1+fix1",
			expectError: false,
		},
		{
			name:        "version with prerelease and existing metadata",
			baseVersion: "1.2.3-rc1+oldfix",
			suffix:      "newfix",
			expected:    "1.2.3-rc1+newfix",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseVer, err := semver.NewVersion(tt.baseVersion)
			if err != nil {
				t.Fatalf("Failed to create base version: %v", err)
			}

			result, err := CreateHotfixVersion(baseVer, tt.suffix)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for suffix %q, but got none", tt.suffix)
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error for suffix %q, but got: %v", tt.suffix, err)
				return
			}

			if result.String() != tt.expected {
				t.Errorf("CreateHotfixVersion(%q, %q) = %q, expected %q", tt.baseVersion, tt.suffix, result.String(), tt.expected)
			}
		})
	}
}

