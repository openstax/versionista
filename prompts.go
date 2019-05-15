package main


import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"io/ioutil"
	"github.com/spf13/viper"
	"github.com/Masterminds/semver"
	"github.com/manifoldco/promptui"
	"github.com/google/go-github/v25/github"
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

func announceVersions(project string, releases []*Release) {
	fmt.Println(
		promptui.Styler(promptui.FGUnderline)(
			fmt.Sprintf("🍀 %s versions are:", project),
		),
	)
	fmt.Println("```")
	for _, release := range releases {
		fmt.Printf("%-20s%s\n",
			release.repository.name,
			release.version.String(),
		)
	}
	aliases := viper.GetStringMapString(fmt.Sprintf("aliases.%s", project))
	for alias, project := range aliases {
		for _, release := range releases {
			if release.repository.String() == project {
				fmt.Printf("%-20s%s\n",
					alias,
					release.version.String(),
				)
				break;
			}
		}


	}
	fmt.Println("```")
}

func announceRelease(repo *Repository, version *semver.Version) {
	fmt.Printf("🎉 released version v%s 🎉\n", version.String())
}

func composeReleaseMessage(cl []ChangeLogEntry) string {
	fpath := os.TempDir() + "/versionista-changelog.txt"
	f, err := os.Create(fpath)
	CheckError(err)

	f.WriteString("### Includes:\n")
	for _, c := range cl {
		f.WriteString(
			fmt.Sprintf(" - #%d %s\n",
				c.Number,
				c.Message,
			),
		)
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

	b, err := ioutil.ReadFile(fpath)
	CheckError(err)
	return string(b)
}

func getVersion(lastVersion *semver.Version, cl []ChangeLogEntry) *semver.Version {

	type option struct {
		Label string
		Version *semver.Version
	}

	fmt.Printf("Last version: %s, %d PR's since then\n", lastVersion.String(), len(cl))
	for _, c := range cl {
		fmt.Printf(" - #%d %20s\n", c.Number, c.Message)
	}
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active: fmt.Sprintf("%s {{ .Label | cyan | underline }} ({{ .Version | green }})", promptui.Styler(promptui.FGGreen)("⇨")),
		Inactive: "  {{ .Label | cyan }} ({{ .Version | green }})",
		Selected: fmt.Sprintf("%s {{ .Label}} to {{ .Version | green | cyan }}", promptui.IconGood),
	}

	major := lastVersion.IncMajor()
	minor := lastVersion.IncMinor()
	patch := lastVersion.IncPatch()

	options := []option {
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
		Items: options,
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
