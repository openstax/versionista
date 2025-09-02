package main

import (
	"testing"

	"github.com/Masterminds/semver"
	
)

func TestBumpChoice(t *testing.T) {
	v := semver.MustParse("1.0.0")
	
	choice := BumpChoice{
		Label:   "Patch",
		Type:    BumpPatch,
		Version: v,
	}
	
	if choice.Label != "Patch" {
		t.Errorf("Expected label 'Patch', got %s", choice.Label)
	}
	
	if choice.Type != BumpPatch {
		t.Errorf("Expected bump type BumpPatch, got %s", choice.Type)
	}
	
	if choice.Version.String() != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", choice.Version.String())
	}
}

func TestPromptFunctionsExist(t *testing.T) {
	// This test just verifies that the functions exist and have the right signatures
	
	// These functions would normally prompt for user input, so we can't call them in tests
	// But we can verify they exist and have the correct signatures
	var f1 func(string, *semver.Version, []Entry) (*semver.Version, BumpType, *HotfixInfo, error) = PromptForVersionBump
	var f3 func(*semver.Version) (string, string, error) = PromptForHotfix
	
	_ = f1
	_ = f3
}