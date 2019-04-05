package main


import (
	"fmt"
	"log"
	"os"
	"bytes"
	"os/exec"
	"github.com/google/go-github/v24/github"
	"github.com/Masterminds/semver"
	"github.com/manifoldco/promptui"
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

func composeReleaseMessage(cl []ChangeLogEntry ) string {
	fpath := os.TempDir() + "/foo.txt"
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

	cmd := exec.Command("vim", fpath)
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

	f.Seek(0, 0)
	buf := new(bytes.Buffer)
	buf.ReadFrom(f)
	return buf.String()
}

func getVersion(lastVersion *semver.Version) *semver.Version {

	type option struct {
		Label string
		Version *semver.Version
	}

	fmt.Printf("Last version: %s", lastVersion.String())

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active: fmt.Sprintf("%s {{ .Label | cyan | underline }} ({{ .Version | green }})", promptui.Styler(promptui.FGGreen)("⇨")),
		Inactive: "  {{ .Label | cyan }} ({{ .Version | green }})",
		Selected: fmt.Sprintf("%s bumping {{ .Label}} to {{ .Version | green | cyan }}", promptui.IconGood),
	}

	major := lastVersion.IncMajor()
	minor := lastVersion.IncMinor()
	patch := lastVersion.IncPatch()

	options := []option {
		{Label: "no, skip release", Version: lastVersion},
		{Label: "Major", Version: &major},
		{Label: "Minor", Version: &minor},
		{Label: "Patch", Version: &patch},
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
