package prompts

import (
	"fmt"
	"os"

	"github.com/Masterminds/semver"
	"github.com/manifoldco/promptui"
	
	"github.com/openstax/versionista/pkg/changelog"
	"github.com/openstax/versionista/pkg/version"
)

type BumpChoice struct {
	Label   string
	Type    version.BumpType
	Version *semver.Version
}

// HotfixInfo holds SHA and suffix for hotfix releases
type HotfixInfo struct {
	SHA    string
	Suffix string
}

// PromptForVersionBump presents an interactive prompt to select version bump type
func PromptForVersionBump(repoName string, lastVersion *semver.Version, entries []changelog.Entry) (*semver.Version, version.BumpType, *HotfixInfo, error) {
	fmt.Printf("\n--- %s ---\n", repoName)
	fmt.Printf("Last version: %s, %d PR's since then\n", version.FormatVersion(lastVersion), len(entries))
	
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

	major := lastVersion.IncMajor()
	minor := lastVersion.IncMinor()
	patch := lastVersion.IncPatch()

	choices := []BumpChoice{
		{Label: "Skip release", Type: version.BumpType("skip"), Version: lastVersion},
		{Label: "Patch", Type: version.BumpPatch, Version: &patch},
		{Label: "Minor", Type: version.BumpMinor, Version: &minor},
		{Label: "Major", Type: version.BumpMajor, Version: &major},
		{Label: "Hotfix", Type: version.BumpHotfix, Version: lastVersion}, // Will be updated after getting suffix
	}

	prompt := promptui.Select{
		Label: fmt.Sprintf(
			"Last version was %s, shall we bump",
			version.FormatVersion(lastVersion),
		),
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
		return nil, version.BumpType("skip"), nil, nil
	}

	// Handle hotfix selection
	if choice.Type == version.BumpHotfix {
		sha, suffix, err := PromptForHotfix(lastVersion)
		if err != nil {
			return nil, "", nil, err
		}

		// Create hotfix version with suffix
		hotfixVersion, err := version.CreateHotfixVersion(lastVersion, suffix)
		if err != nil {
			return nil, "", nil, fmt.Errorf("failed to create hotfix version: %w", err)
		}

		hotfixInfo := &HotfixInfo{
			SHA:    sha,
			Suffix: suffix,
		}

		return hotfixVersion, version.BumpHotfix, hotfixInfo, nil
	}

	return choice.Version, choice.Type, nil, nil
}

// PromptToDelete asks for confirmation to delete a release (for future use)
func PromptToDelete(releaseName string, isDraft bool) (bool, error) {
	templates := &promptui.PromptTemplates{
		Invalid: "{{ . }} was not modified or removed",
		Success: fmt.Sprintf("%s {{ . }} removed", promptui.IconGood),
	}

	var draftLabel = ""
	if isDraft {
		draftLabel = "[DRAFT]"
	}

	prompt := promptui.Prompt{
		Label: fmt.Sprintf("Delete %s Release: %s",
			draftLabel,
			releaseName),
		IsConfirm: true,
		Templates: templates,
	}
	
	result, err := prompt.Run()
	if result == "q" {
		os.Exit(0)
	}
	if err != nil { // no selection, just enter
		return false, nil
	}
	return result == "y", nil
}

// PromptForHotfix asks for SHA and suffix for hotfix releases (for future use)
func PromptForHotfix(lastVersion *semver.Version) (string, string, error) {
	fmt.Printf("Last version: %s\n", version.FormatVersion(lastVersion))

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