package main

import (
	"fmt"
	"strings"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)


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

func releaseSpecifiedProject(cmd *cobra.Command, args []string) {
	var releases []*Release
	eachRepository(args[0], func(repo *Repository) {
		releases = append(releases, cutRelease(repo))
	})
	announceVersions(args[0], releases)
}

func configureCliCommands() {
	var rootCmd = &cobra.Command{
		Short: "versionista",
	}

	releaseCmd := &cobra.Command{
		Use:   "release",
		Short: "release project(s)",
		Args: cobra.MinimumNArgs(1),
		Run: releaseSpecifiedProject,
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
