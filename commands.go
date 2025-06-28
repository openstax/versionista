package main

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strings"
	"sync"
)

func eachRepository(repoSpec string, iterFn func(*Repository)) {
	client := NewClient()
	wg := new(sync.WaitGroup)
	ownerRepo := strings.Split(repoSpec, "/")

	repos := []*Repository{}

	if len(ownerRepo) == 1 {
		projectNames := viper.GetStringSlice(
			fmt.Sprintf("projects.%s", repoSpec),
		)
		for _, name := range projectNames {
			repos = append(repos, NewRepository(name, client))
		}
	} else {
		repos = append(repos, NewRepository(repoSpec, client))
	}

	announceFetching()

	fetchLatest := func(repo *Repository) {
		defer wg.Done()
		repo.resolveVersions(context.Background())
	}

	for _, repo := range repos {
		wg.Add(1)
		go fetchLatest(repo)
	}
	wg.Wait()

	for _, repo := range repos {
		iterFn(repo)
	}

}

func releaseSpecifiedProject(cmd *cobra.Command, args []string) {
	var releases []*Release
	eachRepository(args[0], func(repo *Repository) {
		announceRepo(repo)
		releases = append(releases, newRelease(repo))
	})
	announceVersions(args[0], releases)
}

func hotfixSpecifiedProject(cmd *cobra.Command, args []string) {
	var releases []*Release
	eachRepository(args[0], func(repo *Repository) {
		announceRepo(repo)
		releases = append(releases, newHotfixRelease(repo))
	})
	announceVersions(args[0], releases)
}

func reviewSpecifiedProject(cmd *cobra.Command, args []string) {
	var releases []*Release
	eachRepository(args[0], func(repo *Repository) {
		releases = append(releases, &Release{
			repository: repo,
			version:    repo.latestRelease,
		})
	})
	announceVersions(args[0], releases)
}

func configureCliCommands() {
	var rootCmd = &cobra.Command{
		Short: "versionista",
	}

	rootCmd.AddCommand(&cobra.Command{
		Use:   "release",
		Short: "release project(s)",
		Args:  cobra.MinimumNArgs(1),
		Run:   releaseSpecifiedProject,
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "hotfix",
		Short: "hotfix project(s)",
		Args:  cobra.MinimumNArgs(1),
		Run:   hotfixSpecifiedProject,
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "review",
		Short: "list latest version of project(s)",
		Args:  cobra.MinimumNArgs(1),
		Run:   reviewSpecifiedProject,
	})

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
