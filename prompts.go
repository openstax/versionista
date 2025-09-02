package main

import (
	"fmt"
	"sort"

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

func PromptForVersionBump(repoName string, lastVersion *semver.Version, entries []Entry) (*semver.Version, BumpType, *HotfixInfo, error) {
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
		{Label: "Hotfix", Type: BumpHotfix, Version: lastVersion}, // Will be updated after getting suffix
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
		return nil, "", nil, err
	}

	choice := choices[i]
	if i == 0 { // skip release
		return nil, BumpType("skip"), nil, nil
	}

	// Handle hotfix selection
	if choice.Type == BumpHotfix {
		sha, suffix, err := PromptForHotfix(lastVersion)
		if err != nil {
			return nil, "", nil, err
		}

		// Create hotfix version with suffix
		hotfixVersion, err := CreateHotfixVersion(lastVersion, suffix)
		if err != nil {
			return nil, "", nil, fmt.Errorf("failed to create hotfix version: %w", err)
		}

		hotfixInfo := &HotfixInfo{
			SHA:    sha,
			Suffix: suffix,
		}

		return hotfixVersion, BumpHotfix, hotfixInfo, nil
	}

	return choice.Version, choice.Type, nil, nil
}


func PromptForHotfix(lastVersion *semver.Version) (string, string, error) {
	fmt.Printf("Last version: %s\n", FormatVersion(lastVersion))

	prompt := promptui.Prompt{
		Label: "Enter the SHA for the hotfix",
	}
	sha, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return "", "", err
	}

	prompt = promptui.Prompt{
		Label: "Enter the suffix for the hotfix version",
	}
	suffix, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return "", "", err
	}

	return sha, suffix, nil
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

type ProposedRelease struct {
	Repository     *ReleaseRepository
	HasChanges     bool
	ProposedVersion *semver.Version
	BumpType       BumpType
	Changelog      []Entry
}

type ProjectChoice struct {
	Label     string
	Repository *ReleaseRepository
	Proposed  *ProposedRelease
	Selected  bool
}

func PromptForProjectSelection(proposedChangelogs map[*ReleaseRepository]*ProposedRelease) ([]*ReleaseRepository, error) {
	if len(proposedChangelogs) == 0 {
		return []*ReleaseRepository{}, nil
	}

	// Ensure spinner is completely cleared before displaying changes
	fmt.Print("\r\033[K") // Clear current line completely
	
	// Display all proposed changelogs first
	fmt.Printf("\n=== PROPOSED RELEASES ===\n\n")
	
	// Sort repositories by name for consistent display
	var repos []*ReleaseRepository
	for repo := range proposedChangelogs {
		repos = append(repos, repo)
	}
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].GetDisplayName() < repos[j].GetDisplayName()
	})

	// Display each proposed release
	for _, repo := range repos {
		proposed := proposedChangelogs[repo]
		displayChangelog(repo, proposed)
	}

	// Create choices for selection
	var choices []ProjectChoice
	for _, repo := range repos {
		proposed := proposedChangelogs[repo]
		
		if !proposed.HasChanges {
			continue // Skip repos with no changes
		}

		label := fmt.Sprintf("%s (%s → %s)", 
			repo.GetDisplayName(), 
			FormatVersion(repo.LatestRelease),
			FormatVersion(proposed.ProposedVersion))

		choices = append(choices, ProjectChoice{
			Label:      label,
			Repository: repo,
			Proposed:   proposed,
			Selected:   false,
		})
	}

	if len(choices) == 0 {
		fmt.Printf("No repositories have changes requiring a release.\n")
		return []*ReleaseRepository{}, nil
	}

	// Interactive selection
	fmt.Printf("\n=== SELECT REPOSITORIES TO RELEASE ===\n")
	
	var selectedRepos []*ReleaseRepository
	for _, choice := range choices {

		// Get version bump decision for this repository
		newVersion, bumpType, _, err := PromptForVersionBump(
			choice.Repository.GetDisplayName(), 
			choice.Repository.LatestRelease, 
			choice.Proposed.Changelog)
		if err != nil {
			return nil, err
		}
		
		// Skip if user chose not to release this repository
		if bumpType == BumpType("skip") {
			continue
		}
		
		// Update the proposed release with the selected version
		choice.Proposed.ProposedVersion = newVersion
		choice.Proposed.BumpType = bumpType
		
		selectedRepos = append(selectedRepos, choice.Repository)

	}

	if len(selectedRepos) > 0 {
		fmt.Printf("\nSelected repositories for release:\n")
		for _, repo := range selectedRepos {
			proposed := proposedChangelogs[repo]
			fmt.Printf("- %s: %s → %s\n", 
				repo.GetDisplayName(),
				FormatVersion(repo.LatestRelease),
				FormatVersion(proposed.ProposedVersion))
		}
	}

	return selectedRepos, nil
}

func displayChangelog(repo *ReleaseRepository, proposed *ProposedRelease) {
	fmt.Printf("--- %s ---\n", repo.GetDisplayName())
	
	if !proposed.HasChanges {
		fmt.Printf("No changes since %s\n\n", FormatVersion(repo.LatestRelease))
		return
	}

	if proposed.ProposedVersion != nil {
		fmt.Printf("Current: %s → Proposed: %s (%s)\n", 
			FormatVersion(repo.LatestRelease),
			FormatVersion(proposed.ProposedVersion),
			proposed.BumpType)
	} else {
		fmt.Printf("Current: %s (version bump will be selected interactively)\n", 
			FormatVersion(repo.LatestRelease))
	}
	
	fmt.Printf("%d PR(s) since last release:\n", len(proposed.Changelog))
	for _, entry := range proposed.Changelog {
		fmt.Printf("  • #%d %s (%s)\n", entry.Number, entry.Title, entry.Author)
	}
	fmt.Printf("\n")
}
