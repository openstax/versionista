package main

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/manifoldco/promptui"
)

type BumpChoice struct {
	Label   string
	Type    BumpType
	Version *semver.Version
}

type HotfixInfo struct {
	SHA    string
	Suffix string
}

func PromptForVersionBump(repoName string, lastVersion *semver.Version, entries []Entry) (*semver.Version, BumpType, error) {
	fmt.Printf("\n=== %s ===\n", repoName)
	
	// Check if this is a new project (no previous releases)
	isNewProject := lastVersion.String() == "0.0.0"
	if isNewProject {
		fmt.Printf("No previous releases found, %d PR's for initial release\n", len(entries))
	} else {
		fmt.Printf("Last version: %s, %d PR's since then\n", FormatVersion(lastVersion), len(entries))
	}
	
	// Show recent PRs
	for _, entry := range entries {
		fmt.Printf(" - #%d %s\n", entry.Number, entry.Title)
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   fmt.Sprintf("%s {{ .Label | cyan | underline }} ({{ .Version | green }})", promptui.Styler(promptui.FGGreen)("⇨")),
		Inactive: "  {{ .Label | cyan }} ({{ .Version | green }})",
		Selected: fmt.Sprintf("%s {{ .Label}} to {{ .Version | green | cyan }}", promptui.IconGood),
	}

	major := BumpVersion(lastVersion, BumpMajor)
	minor := BumpVersion(lastVersion, BumpMinor)
	patch := BumpVersion(lastVersion, BumpPatch)

	choices := []BumpChoice{
		{Label: "Skip release", Type: BumpType("skip"), Version: lastVersion},
		{Label: "Patch", Type: BumpPatch, Version: patch},
		{Label: "Minor", Type: BumpMinor, Version: minor},
		{Label: "Major", Type: BumpMajor, Version: major},
	}

	var promptLabel string
	if isNewProject {
		promptLabel = "Create initial release version"
	} else {
		promptLabel = fmt.Sprintf("Last version was %s, shall we bump", FormatVersion(lastVersion))
	}

	prompt := promptui.Select{
		Label:     promptLabel,
		Items:     choices,
		Templates: templates,
	}

	i, _, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return nil, "", err
	}

	choice := choices[i]
	if i == 0 { // skip release
		return nil, BumpType("skip"), nil
	}

	return choice.Version, choice.Type, nil
}


func PromptForHotfixSuffix(lastVersion *semver.Version, sha string) (string, error) {
	fmt.Printf("Last version: %s\n", FormatVersion(lastVersion))
	fmt.Printf("Hotfix SHA: %s\n", sha)

	prompt := promptui.Prompt{
		Label: "Enter the suffix for the hotfix version",
	}
	suffix, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return "", err
	}

	return suffix, nil
}

func PromptToEditChangelog() (bool, error) {
	prompt := promptui.Prompt{
		Label:     "Do you want to edit the changelog before release? (y/N)",
		IsConfirm: true,
	}
	
	result, err := prompt.Run()
	if err != nil {
		// If user just presses enter (no selection), default to no
		if err == promptui.ErrAbort {
			return false, nil
		}
		return false, fmt.Errorf("prompt failed: %w", err)
	}
	
	return result == "y", nil
}




