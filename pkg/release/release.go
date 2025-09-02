package release

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/semver"
	"github.com/google/go-github/v28/github"
	
	"github.com/openstax/versionista/pkg/changelog"
	ghclient "github.com/openstax/versionista/pkg/github"
	"github.com/openstax/versionista/pkg/logging"
	"github.com/openstax/versionista/pkg/prompts"
	"github.com/openstax/versionista/pkg/version"
)

type Type string

const (
	TypeRegular           Type = "regular"
	TypePreReleaseFix     Type = "pre-release-fix"
	TypePostReleaseFix    Type = "post-release-fix"
)

type Manager struct {
	client    *ghclient.Client
	logger    *logging.Logger
	generator *changelog.Generator
}

func NewManager(client *ghclient.Client, logger *logging.Logger, jiraBoards []string) *Manager {
	return &Manager{
		client:    client,
		logger:    logger,
		generator: changelog.NewGenerator(jiraBoards),
	}
}

type Repository struct {
	*ghclient.Repository
	Alias           string
	JiraEnabled     bool
	CrossLinkEnabled bool
	LatestRelease   *semver.Version
	CommitSHA       string
}

func NewRepository(repo *ghclient.Repository, alias string, jiraEnabled, crossLinkEnabled bool, commitSHA string) *Repository {
	return &Repository{
		Repository:      repo,
		Alias:           alias,
		JiraEnabled:     jiraEnabled,
		CrossLinkEnabled: crossLinkEnabled,
		CommitSHA:       commitSHA,
	}
}

func (r *Repository) GetDisplayName() string {
	if r.Alias != "" {
		return r.Alias
	}
	return r.Name
}

type Release struct {
	Repository *Repository
	Version    *semver.Version
	Changelog  []changelog.Entry
}

func (m *Manager) ResolveVersions(ctx context.Context, repo *Repository) error {
	latestRelease, err := m.client.GetLatestRelease(repo.Repository)
	if err != nil {
		v, _ := semver.NewVersion("0.0.0")
		repo.LatestRelease = v
		return nil
	}

	v, err := version.ParseVersion(latestRelease.GetTagName())
	if err != nil {
		return fmt.Errorf("failed to parse latest release version: %w", err)
	}

	repo.LatestRelease = v
	return nil
}

func (m *Manager) HasChanges(ctx context.Context, repo *Repository) (bool, error) {
	// For v0.0.0 (no previous release), check if there are any PRs in the last month
	if repo.LatestRelease.String() == "0.0.0" {
		oneMonthAgo := time.Now().AddDate(0, -1, 0)
		prs, err := m.client.GetRecentMergedPRs(repo.Repository, oneMonthAgo)
		if err != nil {
			// If we can't get recent PRs, assume there are changes to avoid blocking
			m.logger.Debug("Failed to get recent PRs for %s, assuming changes exist: %v", repo.Repository, err)
			return true, nil
		}
		return len(prs) > 0, nil
	}

	// Normal flow - compare with last release
	baseRef := version.FormatVersion(repo.LatestRelease)
	headRef := repo.CommitSHA

	comparison, err := m.client.CompareCommits(repo.Repository, baseRef, headRef)
	if err != nil {
		return false, err
	}

	return len(comparison.Commits) > 0, nil
}

func (m *Manager) GenerateChangelog(ctx context.Context, repo *Repository) ([]changelog.Entry, error) {
	var entries []changelog.Entry
	
	// Check if this is a fresh repository (v0.0.0) - use PRs from last month
	if repo.LatestRelease.String() == "0.0.0" {
		m.logger.Debug("No previous releases found for %s, using PRs from last month", repo.Repository)
		
		oneMonthAgo := time.Now().AddDate(0, -1, 0)
		prs, err := m.client.GetRecentMergedPRs(repo.Repository, oneMonthAgo)
		if err != nil {
			return nil, fmt.Errorf("failed to get recent PRs: %w", err)
		}

		for _, pr := range prs {
			entry := changelog.Entry{
				Number:      pr.GetNumber(),
				Date:        pr.GetMergedAt().Format("2006-01-02"),
				Author:      pr.GetUser().GetLogin(),
				Title:       pr.GetTitle(),
				Description: pr.GetBody(),
			}

			if repo.JiraEnabled {
				tickets := m.generator.ExtractTickets(pr.GetTitle())
				entry.Tickets = append(entry.Tickets, tickets...)
			}

			entries = append(entries, entry)
		}
		
		return entries, nil
	}

	// Normal flow - compare with last release
	baseRef := version.FormatVersion(repo.LatestRelease)
	headRef := repo.CommitSHA

	comparison, err := m.client.CompareCommits(repo.Repository, baseRef, headRef)
	if err != nil {
		return nil, err
	}

	for _, commit := range comparison.Commits {
		prNumber, err := changelog.ParsePRNumber(commit.GetCommit().GetMessage())
		if err != nil {
			m.logger.Debug("Skipping commit %s: %v", commit.GetSHA()[:8], err)
			continue
		}

		pr, err := m.client.GetPullRequest(repo.Repository, prNumber)
		if err != nil {
			m.logger.Debug("Failed to get PR #%d: %v", prNumber, err)
			continue
		}

		entry := changelog.Entry{
			Number:      prNumber,
			Date:        pr.GetMergedAt().Format("2006-01-02"),
			Author:      pr.GetUser().GetLogin(),
			Title:       pr.GetTitle(),
			Description: pr.GetBody(),
		}

		if repo.JiraEnabled {
			tickets := m.generator.ExtractTickets(pr.GetTitle())
			entry.Tickets = append(entry.Tickets, tickets...)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func (m *Manager) CreateRelease(ctx context.Context, repo *Repository, newVersion *semver.Version, 
	entries []changelog.Entry, crossLinks []changelog.CrossLink, releaseType Type) error {
	
	releaseNotes := changelog.BuildReleaseNotes(entries, crossLinks)
	tagName := version.FormatVersion(newVersion)
	
	isDraft := false

	release := &github.RepositoryRelease{
		TagName:    &tagName,
		Name:       &tagName,
		Body:       &releaseNotes,
		Draft:      &isDraft,
	}

	_, err := m.client.CreateRelease(repo.Repository, release)
	if err != nil {
		return fmt.Errorf("failed to create release: %w", err)
	}

	m.logger.Info("Successfully created release %s for %s", tagName, repo.Repository)
	return nil
}

func (m *Manager) CreateHotfixRelease(ctx context.Context, repo *Repository, newVersion *semver.Version, 
	entries []changelog.Entry, crossLinks []changelog.CrossLink, releaseType Type, targetSHA string) error {
	
	releaseNotes := changelog.BuildReleaseNotes(entries, crossLinks)
	tagName := version.FormatVersion(newVersion)
	
	isDraft := false


	release := &github.RepositoryRelease{
		TagName:    &tagName,
		Name:       &tagName,
		Body:       &releaseNotes,
		Draft:      &isDraft,
	}

	_, err := m.client.CreateReleaseFromSHA(repo.Repository, release, targetSHA)
	if err != nil {
		return fmt.Errorf("failed to create hotfix release: %w", err)
	}

	m.logger.Info("Successfully created hotfix release %s for %s from SHA %s", tagName, repo.Repository, targetSHA)
	return nil
}

func (m *Manager) ProcessRelease(ctx context.Context, repo *Repository, releaseType Type, 
	bumpType version.BumpType, allRepos []*Repository) (*Release, error) {
	
	if err := m.ResolveVersions(ctx, repo); err != nil {
		return nil, fmt.Errorf("failed to resolve versions: %w", err)
	}

	hasChanges, err := m.HasChanges(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to check for changes: %w", err)
	}

	if !hasChanges {
		m.logger.Info("No changes found for %s since %s", repo.Repository, version.FormatVersion(repo.LatestRelease))
		return &Release{
			Repository: repo,
			Version:    repo.LatestRelease,
			Changelog:  []changelog.Entry{},
		}, nil
	}

	entries, err := m.GenerateChangelog(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to generate changelog: %w", err)
	}

	newVersion := version.BumpVersion(repo.LatestRelease, bumpType)

	var crossLinks []changelog.CrossLink
	if repo.CrossLinkEnabled && len(allRepos) > 1 {
		crossLinks = m.generateCrossLinks(repo, allRepos, newVersion)
	}

	if err := m.CreateRelease(ctx, repo, newVersion, entries, crossLinks, releaseType); err != nil {
		return nil, err
	}

	return &Release{
		Repository: repo,
		Version:    newVersion,
		Changelog:  entries,
	}, nil
}

func (m *Manager) ProcessReleaseInteractive(ctx context.Context, repo *Repository, releaseType Type, allRepos []*Repository) (*Release, error) {
	if err := m.ResolveVersions(ctx, repo); err != nil {
		return nil, fmt.Errorf("failed to resolve versions: %w", err)
	}

	hasChanges, err := m.HasChanges(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to check for changes: %w", err)
	}

	if !hasChanges {
		m.logger.Info("No changes found for %s since %s", repo.Repository, version.FormatVersion(repo.LatestRelease))
		return &Release{
			Repository: repo,
			Version:    repo.LatestRelease,
			Changelog:  []changelog.Entry{},
		}, nil
	}

	entries, err := m.GenerateChangelog(ctx, repo)
	if err != nil {
		m.logger.Warn("Failed to generate changelog for %s: %v", repo.Repository, err)
		m.logger.Info("Proceeding with interactive prompt without changelog data")
		entries = []changelog.Entry{} // Use empty changelog
	}

	// Interactive prompt for version bump
	repoDisplayName := repo.GetDisplayName()
	newVersion, bumpType, hotfixInfo, err := prompts.PromptForVersionBump(repoDisplayName, repo.LatestRelease, entries)
	if err != nil {
		return nil, fmt.Errorf("failed to get version bump choice: %w", err)
	}

	// If user chose to skip release
	if bumpType == "skip" {
		m.logger.Info("Skipping release for %s", repoDisplayName)
		return &Release{
			Repository: repo,
			Version:    repo.LatestRelease,
			Changelog:  entries,
		}, nil
	}

	var crossLinks []changelog.CrossLink
	if repo.CrossLinkEnabled && len(allRepos) > 1 {
		crossLinks = m.generateCrossLinks(repo, allRepos, newVersion)
	}

	// Create the appropriate type of release
	if bumpType == version.BumpHotfix && hotfixInfo != nil {
		m.logger.Info("Creating hotfix release for %s: %s (SHA: %s)", repoDisplayName, version.FormatVersion(newVersion), hotfixInfo.SHA)
		if err := m.CreateHotfixRelease(ctx, repo, newVersion, entries, crossLinks, releaseType, hotfixInfo.SHA); err != nil {
			return nil, err
		}
	} else {
		if err := m.CreateRelease(ctx, repo, newVersion, entries, crossLinks, releaseType); err != nil {
			return nil, err
		}
	}

	return &Release{
		Repository: repo,
		Version:    newVersion,
		Changelog:  entries,
	}, nil
}

func (m *Manager) generateCrossLinks(currentRepo *Repository, allRepos []*Repository, currentVersion *semver.Version) []changelog.CrossLink {
	var links []changelog.CrossLink

	for _, repo := range allRepos {
		// Skip the current repository - don't include it in its own cross-links
		if repo == currentRepo {
			continue
		}

		var repoName string
		if repo.Alias != "" {
			repoName = repo.Alias
		} else {
			repoName = repo.Name
		}

		releaseURL := fmt.Sprintf("https://github.com/%s/%s/releases/tag/v%s",
			repo.Owner, repo.Name, repo.LatestRelease.String())

		links = append(links, changelog.CrossLink{
			Name:    repoName,
			Version: repo.LatestRelease.String(),
			URL:     releaseURL,
		})
	}

	return links
}
