package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/google/go-github/v28/github"
	
)

type Type string

const (
	TypeRegular           Type = "regular"
	TypePreReleaseFix     Type = "pre-release-fix"
	TypePostReleaseFix    Type = "post-release-fix"
)

type Manager struct {
	client    *Client
	logger    *Logger
	generator *Generator
	jiraOrgId string
	dryRun    bool
}

func NewManager(client *Client, logger *Logger, jiraBoards []string, jiraOrgId string, dryRun bool) *Manager {
	return &Manager{
		client:    client,
		logger:    logger,
		generator: NewGenerator(jiraBoards),
		jiraOrgId: jiraOrgId,
		dryRun:    dryRun,
	}
}

type ReleaseRepository struct {
	*Repository
	Alias           string
	JiraEnabled     bool
	CrossLinkEnabled bool
	LatestRelease   *semver.Version
	CommitSHA       string
}

func NewRepository(repo *Repository, alias string, jiraEnabled, crossLinkEnabled bool, commitSHA string) *ReleaseRepository {
	return &ReleaseRepository{
		Repository:      repo,
		Alias:           alias,
		JiraEnabled:     jiraEnabled,
		CrossLinkEnabled: crossLinkEnabled,
		CommitSHA:       commitSHA,
	}
}

func (r *ReleaseRepository) GetDisplayName() string {
	if r.Alias != "" {
		return r.Alias
	}
	return r.Name
}

type Release struct {
	Repository *ReleaseRepository
	Version    *semver.Version
	Changelog  []Entry
}

func (m *Manager) ResolveVersions(ctx context.Context, repo *ReleaseRepository) error {
	latestRelease, err := m.client.GetLatestRelease(repo.Repository)
	if err != nil {
		v, _ := semver.NewVersion("0.0.0")
		repo.LatestRelease = v
		return nil
	}

	v, err := ParseVersion(latestRelease.GetTagName())
	if err != nil {
		return fmt.Errorf("failed to parse latest release version: %w", err)
	}

	repo.LatestRelease = v
	return nil
}

func (m *Manager) HasChanges(ctx context.Context, repo *ReleaseRepository) (bool, error) {
	// For v0.0.0 (no previous release), check if there are any merged PRs (last 10)
	if repo.LatestRelease.String() == "0.0.0" {
		prs, err := m.client.GetLastNMergedPRs(repo.Repository, 10)
		if err != nil {
			// If we can't get recent PRs, assume there are changes to avoid blocking
			m.logger.Debug("Failed to get last 10 PRs for %s, assuming changes exist: %v", repo.Repository, err)
			return true, nil
		}
		return len(prs) > 0, nil
	}

	// Normal flow - compare with last release
	baseRef := FormatVersion(repo.LatestRelease)
	headRef := repo.CommitSHA

	comparison, err := m.client.CompareCommits(repo.Repository, baseRef, headRef)
	if err != nil {
		return false, err
	}

	return len(comparison.Commits) > 0, nil
}

func (m *Manager) GenerateChangelog(ctx context.Context, repo *ReleaseRepository) ([]Entry, error) {
	return m.GenerateChangelogFromSHA(ctx, repo, "")
}

func (m *Manager) GenerateChangelogFromSHA(ctx context.Context, repo *ReleaseRepository, targetSHA string) ([]Entry, error) {
	prs, err := m.getPRsForChangelog(ctx, repo, targetSHA)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, pr := range prs {
		entry := m.createEntryFromPR(repo, pr)
		entries = append(entries, entry)
	}

	return entries, nil
}

func (m *Manager) getPRsForChangelog(ctx context.Context, repo *ReleaseRepository, targetSHA string) ([]*github.PullRequest, error) {
	// Check if this is a fresh repository (v0.0.0) - use last 10 PRs
	if repo.LatestRelease.String() == "0.0.0" {
		m.logger.Debug("No previous releases found for %s, using last 10 PRs", repo.Repository)
		
		prs, err := m.client.GetLastNMergedPRs(repo.Repository, 10)
		if err != nil {
			return nil, fmt.Errorf("failed to get last 10 PRs: %w", err)
		}
		return prs, nil
	}

	// Determine the head reference
	var headRef string
	if targetSHA != "" {
		headRef = targetSHA
		m.logger.Debug("Using target SHA for changelog generation: %s", targetSHA)
	} else {
		headRef = repo.CommitSHA
	}

	// Compare with last release
	baseRef := FormatVersion(repo.LatestRelease)
	comparison, err := m.client.CompareCommits(repo.Repository, baseRef, headRef)
	if err != nil {
		return nil, err
	}

	var prs []*github.PullRequest
	for _, commit := range comparison.Commits {
		prNumber, err := ParsePRNumber(commit.GetCommit().GetMessage())
		if err != nil {
			continue
		}

		pr, err := m.client.GetPullRequest(repo.Repository, prNumber)
		if err != nil {
			continue
		}

		prs = append(prs, pr)
	}

	return prs, nil
}

func (m *Manager) createEntryFromPR(repo *ReleaseRepository, pr *github.PullRequest) Entry {
	entry := Entry{
		Number:      pr.GetNumber(),
		Date:        pr.GetMergedAt().Format("2006-01-02"),
		Author:      pr.GetUser().GetLogin(),
		Title:       pr.GetTitle(),
		Description: pr.GetBody(),
	}

	if repo.JiraEnabled {
		entry.Tickets = m.extractTicketsFromPR(repo, pr)
	}

	return entry
}

func (m *Manager) extractTicketsFromPR(repo *ReleaseRepository, pr *github.PullRequest) []string {
	var allText []string

	// Collect title
	if pr.GetTitle() != "" {
		allText = append(allText, pr.GetTitle())
	}
	
	// Collect body
	if pr.GetBody() != "" {
		allText = append(allText, pr.GetBody())
	}

	// Collect comments
	comments, err := m.client.GetPullRequestComments(repo.Repository, pr.GetNumber())
	if err != nil {
		m.logger.Debug("Failed to get comments for PR #%d: %v", pr.GetNumber(), err)
	} else {
		for _, comment := range comments {
			if comment.GetBody() != "" {
				allText = append(allText, comment.GetBody())
			}
		}
	}

	// Combine all text parts and extract tickets once
	combinedText := strings.Join(allText, " ")
	tickets := m.generator.ExtractTickets(combinedText)

	m.logger.Debug("Found %d tickets for %d\n%s\n", len(tickets), pr.GetNumber(), allText)
	
	// Remove any duplicates
	return removeDuplicates(tickets)
}

func (m *Manager) CreateRelease(ctx context.Context, repo *ReleaseRepository, newVersion *semver.Version, 
	releaseNotes string, releaseType Type) error {
	
	tagName := FormatVersion(newVersion)
	isDraft := false

	if m.dryRun {
		m.logger.Info("[DRY RUN] Would create release %s for %s", tagName, repo.Repository)
		m.logger.Debug("[DRY RUN] Release notes:\n%s", releaseNotes)
		return nil
	}

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

func (m *Manager) CreateReleaseFromEntries(ctx context.Context, repo *ReleaseRepository, newVersion *semver.Version,
	entries []Entry, crossLinks []CrossLink, releaseType Type) error {

	releaseNotes := BuildReleaseNotes(entries, crossLinks, repo.JiraEnabled, m.jiraOrgId)
	return m.CreateRelease(ctx, repo, newVersion, releaseNotes, releaseType)
}

func (m *Manager) ProcessRelease(ctx context.Context, repo *ReleaseRepository, releaseType Type, 
	bumpType BumpType, allRepos []*ReleaseRepository) (*Release, error) {
	
	if err := m.ResolveVersions(ctx, repo); err != nil {
		return nil, fmt.Errorf("failed to resolve versions: %w", err)
	}

	hasChanges, err := m.HasChanges(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to check for changes: %w", err)
	}

	if !hasChanges {
		m.logger.Info("No changes found for %s since %s", repo.Repository, FormatVersion(repo.LatestRelease))
		return &Release{
			Repository: repo,
			Version:    repo.LatestRelease,
			Changelog:  []Entry{},
		}, nil
	}

	entries, err := m.GenerateChangelog(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to generate changelog: %w", err)
	}

	newVersion := BumpVersion(repo.LatestRelease, bumpType)

	var crossLinks []CrossLink
	if repo.CrossLinkEnabled && len(allRepos) > 1 {
		crossLinks = m.generateCrossLinks(repo, allRepos, newVersion)
	}

	if err := m.CreateReleaseFromEntries(ctx, repo, newVersion, entries, crossLinks, releaseType); err != nil {
		return nil, err
	}

	return &Release{
		Repository: repo,
		Version:    newVersion,
		Changelog:  entries,
	}, nil
}

func (m *Manager) PrepareReleaseData(ctx context.Context, repo *ReleaseRepository) ([]Entry, error) {
	if err := m.ResolveVersions(ctx, repo); err != nil {
		return nil, fmt.Errorf("failed to resolve versions: %w", err)
	}

	hasChanges, err := m.HasChanges(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to check for changes: %w", err)
	}

	if !hasChanges {
		return []Entry{}, nil // No changes, return empty changelog
	}

	entries, err := m.GenerateChangelog(ctx, repo)
	if err != nil {
		m.logger.Warn("Failed to generate changelog for %s: %v", repo.Repository, err)
		m.logger.Info("Proceeding with interactive prompt without changelog data")
		entries = []Entry{} // Use empty changelog
	}

	return entries, nil
}

func (m *Manager) ProcessReleaseInteractive(ctx context.Context, repo *ReleaseRepository, releaseType Type, allRepos []*ReleaseRepository) (*Release, error) {
	entries, err := m.PrepareReleaseData(ctx, repo)
	if err != nil {
		return nil, err
	}
	return m.ProcessReleaseInteractiveWithEntries(ctx, repo, releaseType, allRepos, entries)
}

func (m *Manager) ProcessReleaseInteractiveWithEntries(ctx context.Context, repo *ReleaseRepository, releaseType Type, allRepos []*ReleaseRepository, entries []Entry) (*Release, error) {
	// Check if no changes
	if len(entries) == 0 {
		hasChanges, err := m.HasChanges(ctx, repo)
		if err != nil {
			return nil, fmt.Errorf("failed to check for changes: %w", err)
		}
		
		if !hasChanges {
			m.logger.Info("No changes found for %s since %s", repo.Repository, FormatVersion(repo.LatestRelease))
			return &Release{
				Repository: repo,
				Version:    repo.LatestRelease,
				Changelog:  []Entry{},
			}, nil
		}
	}

	// Interactive prompt for version bump
	repoDisplayName := repo.GetDisplayName()
	newVersion, bumpType, err := PromptForVersionBump(repoDisplayName, repo.LatestRelease, entries)
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

	var crossLinks []CrossLink
	if repo.CrossLinkEnabled && len(allRepos) > 1 {
		crossLinks = m.generateCrossLinks(repo, allRepos, newVersion)
	}

	// Ask if user wants to edit changelog
	var releaseNotes string

	if len(entries) > 0 {
		wantEdit, err := PromptToEditChangelog()
		if err != nil {
			m.logger.Warn("Failed to prompt for changelog editing: %v", err)
			releaseNotes = BuildReleaseNotes(entries, crossLinks, repo.JiraEnabled, m.jiraOrgId)
		} else if wantEdit {
			m.logger.Info("Opening changelog for editing...")
			editedText, err := EditChangelog(entries, crossLinks, repo.JiraEnabled, m.jiraOrgId)
			if err != nil {
				m.logger.Warn("Failed to edit changelog: %v", err)
				m.logger.Info("Using original changelog")
				releaseNotes = BuildReleaseNotes(entries, crossLinks, repo.JiraEnabled, m.jiraOrgId)
			} else {
				releaseNotes = editedText
				m.logger.Info("Using edited changelog")
			}
		} else {
			releaseNotes = BuildReleaseNotes(entries, crossLinks, repo.JiraEnabled, m.jiraOrgId)
		}
	} else {
		releaseNotes = BuildReleaseNotes(entries, crossLinks, repo.JiraEnabled, m.jiraOrgId)
	}

	// Create the release
	if err := m.CreateRelease(ctx, repo, newVersion, releaseNotes, releaseType); err != nil {
		return nil, err
	}

	return &Release{
		Repository: repo,
		Version:    newVersion,
		Changelog:  entries,
	}, nil
}



func (m *Manager) generateCrossLinks(currentRepo *ReleaseRepository, repos []*ReleaseRepository, currentVersion *semver.Version) []CrossLink {
	var links []CrossLink

	for _, repo := range repos {
		// Skip the current repository - don't include it in its own cross-links
		if repo == currentRepo {
			continue
		}

		// Use the latest release version for cross-links
		version := repo.LatestRelease

		releaseURL := fmt.Sprintf("https://github.com/%s/%s/releases/tag/v%s",
			repo.Owner, repo.Name, version.String())

		links = append(links, CrossLink{
			Name:    repo.GetDisplayName(),
			Version: version.String(),
			URL:     releaseURL,
		})
	}

	return links
}

