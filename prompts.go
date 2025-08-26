package main

import (
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/google/go-github/v28/github"
	"github.com/manifoldco/promptui"
	"github.com/spf13/viper"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	ReleaseTypeVersion        = "release"
	ReleaseTypePostReleaseFix = "post-release"
	ReleaseTypePreReleaseFix  = "pre-release"
)

func promptToDelete(release *github.RepositoryRelease) bool {
	templates := &promptui.PromptTemplates{
		Invalid: "{{ . }} was not modified or removed",
		Success: fmt.Sprintf("%s {{ . }} removed", promptui.IconGood),
	}

	var draftLabel = ""
	if *release.Draft {
		draftLabel = "[DRAFT]"
	}

	prompt := promptui.Prompt{
		Label: fmt.Sprintf("Delete %s Release: %s",
			draftLabel,
			release.GetName()),
		IsConfirm: true,
		Templates: templates,
	}
	result, err := prompt.Run()
	if result == "q" {
		os.Exit(0)
	}
	if err != nil { // no selection, just enter
		return false
	}
	return result == "y"
}

func announceRepo(repo *Repository) {
	fmt.Println(
		promptui.Styler(promptui.BGBlue)(repo.String()),
	)
}

func announceFetching() {
	fmt.Println(
		promptui.Styler(promptui.FGGreen)("Fetching repo info…"),
	)
}

const version_format = "%-24s%10s\n"

func announceVersions(project string, releases []*Release) {
	fmt.Println(
		promptui.Styler(promptui.FGUnderline)(
			fmt.Sprintf("%s versions are:", project),
		),
	)
	fmt.Println("```")

	for _, release := range releases {
		var name = release.repository.name

		// Use the alias from the repository config if available
		if release.repository.alias != "" {
			name = release.repository.alias
		}
		fmt.Printf(version_format,
			name,
			fmt.Sprintf("v%s", release.version.String()),
		)
	}
	fmt.Println("```")
}

func announceRelease(repo *Repository, version *semver.Version) {
	fmt.Printf("🎉 released version v%s 🎉\n", version.String())
}

func composeReleaseMessage(cl []ChangeLogEntry, currentRepo *Repository, allRepos []*Repository, currentVersion *semver.Version) string {
	jiraSlug := viper.GetString("jira_slug")
	fpath := os.TempDir() + "/versionista-changelog.txt"
	f, err := os.Create(fpath)
	CheckError(err)

	// Add cross-links if enabled for this repository
	if currentRepo.crossLinkEnabled && len(allRepos) > 1 {
		f.WriteString("## Related Releases\n\n")
		for _, repo := range allRepos {
			var repoName string
			if repo.alias != "" {
				repoName = repo.alias
			} else {
				repoName = repo.name
			}
			
			var version *semver.Version
			if repo == currentRepo {
				// Use the version being released for the current repo
				version = currentVersion
			} else {
				// Use the latest release version for other repos
				version = repo.latestRelease
			}
			
			releaseURL := fmt.Sprintf("https://github.com/%s/%s/releases/tag/v%s", 
				repo.owner, repo.name, version.String())
			f.WriteString(fmt.Sprintf("- [%s v%s](%s)\n", repoName, version.String(), releaseURL))
		}
		f.WriteString("\n---\n\n")
	}

	for _, c := range cl {
		var tickets []string
		for _, ticket := range c.Tickets {
			tickets = append(tickets, fmt.Sprintf("[%s](https://%s.atlassian.net/browse/%s)", ticket, jiraSlug, ticket))
		}
		
		// Write the summary with PR title
		f.WriteString(fmt.Sprintf("<summary>#%d: %s</summary>\n\n", c.Number, c.Title))
		
		// Write the details section with PR information
		f.WriteString("<details>\n\n")
		f.WriteString(fmt.Sprintf("**Author:** %s  \n", c.Author))
		f.WriteString(fmt.Sprintf("**Merged Date:** %s  \n", c.Date))
		if len(tickets) > 0 {
			f.WriteString(fmt.Sprintf("**Internal Ticket #:** %s  \n", strings.Join(tickets, ", ")))
		}
		if c.Description != "" {
			f.WriteString(fmt.Sprintf("\n**Description:**\n%s\n", c.Description))
		}
		f.WriteString("\n</details>\n\n")
	}
	f.Close()
	var editor = viper.GetString("editor")
	if editor == "" {
		editor = "vim"
	}
	cmd := exec.Command(editor, fpath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		log.Printf("2")
		log.Fatal(err)
	}
	err = cmd.Wait()
	CheckError(err)

	b, err := os.ReadFile(fpath)
	CheckError(err)
	return string(b)
}

func getReleaseType() string {
	prompt := promptui.Select{
		Label: "Select release type",
		Items: []string{ReleaseTypeVersion, ReleaseTypePostReleaseFix, ReleaseTypePreReleaseFix},
	}
	_, result, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return ""
	}
	return result
}

func getVersion(lastVersion *semver.Version, cl []ChangeLogEntry) *semver.Version {

	type option struct {
		Label   string
		Version *semver.Version
	}

	fmt.Printf("Last version: %s, %d PR's since then\n", lastVersion.String(), len(cl))
	for _, c := range cl {
		fmt.Printf(" - #%d %20s\n", c.Number, c.Title)
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

	options := []option{
		{Label: "skip release", Version: lastVersion},
		{Label: "Patch", Version: &patch},
		{Label: "Minor", Version: &minor},
		{Label: "Major", Version: &major},
	}

	prompt := promptui.Select{
		Label: fmt.Sprintf(
			"Last version was %s, shall we bump",
			lastVersion.String(),
		),
		Items:     options,
		Templates: templates,
	}
	i, _, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return nil
	}
	choice := options[i]
	if i == 0 { // no release
		return nil
	}
	return choice.Version

}

func getPreReleaseFixInfo(repo *Repository, lastVersion *semver.Version) (*semver.Version, string) {
	fmt.Printf("Last version: %s\n", lastVersion.String())

	prompt := promptui.Prompt{
		Label:   "Enter the SHA for the pre-release",
		Default: repo.GetLatestSHA("main"),
	}
	sha, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return nil, ""
	}

	prompt = promptui.Prompt{
		Label: "Enter the suffix for the pre-release",
	}
	suffix, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return nil, ""
	}

	version, err := lastVersion.SetPrerelease(suffix)
	if err != nil {
		fmt.Printf("Failed to set pre-release version: %v\n", err)
		return nil, ""
	}

	return &version, sha
}

func getPostReleaseFixInfo(repo *Repository, lastVersion *semver.Version) (*semver.Version, string) {
	fmt.Printf("Last version: %s\n", lastVersion.String())

	prompt := promptui.Prompt{
		Label:   "Enter the SHA for the post-release",
		Default: repo.GetLatestSHA("main"),
	}
	sha, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return nil, ""
	}

	prompt = promptui.Prompt{
		Label: "Enter the suffix for the post-release version",
	}
	suffix, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return nil, ""
	}

	version, err := lastVersion.SetMetadata(suffix)
	if err != nil {
		fmt.Printf("Failed to set post-release version: %v\n", err)
		return nil, ""
	}

	return &version, sha
}


