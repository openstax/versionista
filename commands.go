package main

import (
	"fmt"
	"strings"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func cutRelease(repo *Repository) {
	lastRelease := repo.latestRelease()
	changeLog := repo.getChangelog(lastRelease)
	if 0 == len(changeLog) {
		fmt.Printf("  skipping, no PRs found since %s\n", lastRelease.String())
	} else {
		newVersion := getVersion(lastRelease)
		if newVersion != nil {
			msg := composeReleaseMessage(changeLog)
			repo.createRelease(newVersion, msg)
		}
		announceRelease(repo, repo.latestRelease());
	}
}

func eachRepository(repoSpec string, iterFn func(*Repository)) {
	client := NewClient()
	ownerRepo := strings.Split(repoSpec, "/")
	announceAndCall := func(name string) {
		repo := NewRepository(name, client)
		announceRepo(repo);
		iterFn(repo)
	}
	if len(ownerRepo) == 1 {
		projectNames := viper.GetStringSlice(
			fmt.Sprintf("projects.%s", repoSpec),
		)
		for _, name := range(projectNames) {
			announceAndCall(name)
		}
	} else {
		announceAndCall(repoSpec)
	}
}

func configureCliCommands() {
	var rootCmd = &cobra.Command{
		Short: "versionista",
	}

	releaseCmd := &cobra.Command{
		Use:   "release",
		Short: "release project(s)",
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			eachRepository(args[0], func(repo *Repository) {
				cutRelease(repo)
			})
		},
	}
	rootCmd.AddCommand(releaseCmd)

	////////////////////////////////
	// no mass deleting releases  //
	////////////////////////////////

	// deleteCmd := &cobra.Command{
	//	Use:   "delete",
	//	Short: "delete [thing]",
	// }
	// rootCmd.AddCommand(deleteCmd)
	// deleteCmd.AddCommand(&cobra.Command{
	//	Use:   "releases",
	//	Short: "release",
	//	Args: cobra.MinimumNArgs(1),
	//	Run: func(cmd *cobra.Command, args []string) {
	//		eachRepository(args[0], func(repo *Repository)  {
	//			for _, release := range repo.getRecentReleases() {
	//				if promptToDelete(release) {
	//					repo.deleteRelease(release)
	//				}
	//			}
	//		})
	//	}})

	rootCmd.Execute()

}
