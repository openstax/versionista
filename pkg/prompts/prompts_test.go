package prompts

import (
	"testing"

	"github.com/Masterminds/semver"
	
	"github.com/openstax/versionista/pkg/changelog"
	"github.com/openstax/versionista/pkg/version"
)

func TestBumpChoice(t *testing.T) {
	v := semver.MustParse("1.0.0")
	
	choice := BumpChoice{
		Label:   "Patch",
		Type:    version.BumpPatch,
		Version: v,
	}
	
	if choice.Label != "Patch" {
		t.Errorf("Expected label 'Patch', got %s", choice.Label)
	}
	
	if choice.Type != version.BumpPatch {
		t.Errorf("Expected bump type BumpPatch, got %s", choice.Type)
	}
	
	if choice.Version.String() != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", choice.Version.String())
	}
}

// Note: The interactive prompt functions cannot be easily tested without mocking promptui
// These functions would typically be tested with integration tests or by mocking the promptui library
func TestPromptFunctionsExist(t *testing.T) {
	// This test just verifies that the functions exist and have the right signatures
	
	// These functions would normally prompt for user input, so we can't call them in tests
	// But we can verify they exist and have the correct signatures
	var f1 func(string, *semver.Version, []changelog.Entry) (*semver.Version, version.BumpType, *HotfixInfo, error) = PromptForVersionBump
	var f2 func(string, bool) (bool, error) = PromptToDelete
	var f3 func(*semver.Version) (string, string, error) = PromptForHotfix
	
	_ = f1
	_ = f2
	_ = f3
}