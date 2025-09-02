package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v28/github"
	"golang.org/x/oauth2"
)

type Client struct {
	*github.Client
	ctx context.Context
}

func NewClient(token string) *Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &Client{
		Client: github.NewClient(tc),
		ctx:    ctx,
	}
}

type Repository struct {
	Owner string
	Name  string
}

func ParseRepoSpec(repoSpec string) (*Repository, error) {
	parts := strings.Split(repoSpec, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository specification: %s (expected format: owner/repo)", repoSpec)
	}

	return &Repository{
		Owner: parts[0],
		Name:  parts[1],
	}, nil
}

func (r *Repository) String() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

func (c *Client) GetLatestRelease(repo *Repository) (*github.RepositoryRelease, error) {
	release, _, err := c.Repositories.GetLatestRelease(c.ctx, repo.Owner, repo.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release for %s: %w", repo, err)
	}
	return release, nil
}

func (c *Client) GetReleases(repo *Repository) ([]*github.RepositoryRelease, error) {
	releases, _, err := c.Repositories.ListReleases(c.ctx, repo.Owner, repo.Name, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get releases for %s: %w", repo, err)
	}
	return releases, nil
}

func (c *Client) CompareCommits(repo *Repository, base, head string) (*github.CommitsComparison, error) {
	comparison, _, err := c.Repositories.CompareCommits(c.ctx, repo.Owner, repo.Name, base, head)
	if err != nil {
		return nil, fmt.Errorf("failed to compare commits %s...%s for %s: %w", base, head, repo, err)
	}
	return comparison, nil
}

func (c *Client) GetPullRequest(repo *Repository, number int) (*github.PullRequest, error) {
	pr, _, err := c.PullRequests.Get(c.ctx, repo.Owner, repo.Name, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request #%d for %s: %w", number, repo, err)
	}
	return pr, nil
}

func (c *Client) GetRecentMergedPRs(repo *Repository, since time.Time) ([]*github.PullRequest, error) {
	opts := &github.PullRequestListOptions{
		State:     "closed",
		Sort:      "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}

	var allPRs []*github.PullRequest
	
	for {
		prs, resp, err := c.PullRequests.List(c.ctx, repo.Owner, repo.Name, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get pull requests for %s: %w", repo, err)
		}

		for _, pr := range prs {
			// Only include merged PRs
			if pr.GetMergedAt().IsZero() {
				continue
			}
			
			// Stop if we've gone past our since date
			if pr.GetMergedAt().Before(since) {
				return allPRs, nil
			}
			
			allPRs = append(allPRs, pr)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allPRs, nil
}

func (c *Client) CreateRelease(repo *Repository, release *github.RepositoryRelease) (*github.RepositoryRelease, error) {
	createdRelease, _, err := c.Repositories.CreateRelease(c.ctx, repo.Owner, repo.Name, release)
	if err != nil {
		return nil, fmt.Errorf("failed to create release for %s: %w", repo, err)
	}
	return createdRelease, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *Client) GetLastNMergedPRs(repo *Repository, count int) ([]*github.PullRequest, error) {
	opts := &github.PullRequestListOptions{
		State:     "closed",
		Sort:      "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: minInt(count, 100), // GitHub API limit is 100 per page
		},
	}

	var allPRs []*github.PullRequest
	needed := count
	
	for needed > 0 {
		prs, resp, err := c.PullRequests.List(c.ctx, repo.Owner, repo.Name, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get pull requests for %s: %w", repo, err)
		}

		for _, pr := range prs {
			// Only include merged PRs
			if pr.GetMergedAt().IsZero() {
				continue
			}
			
			allPRs = append(allPRs, pr)
			needed--
			
			if needed == 0 {
				return allPRs, nil
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
		// Update PerPage for remaining items
		opts.PerPage = minInt(needed, 100)
	}

	return allPRs, nil
}

func (c *Client) CreateReleaseFromSHA(repo *Repository, release *github.RepositoryRelease, targetCommitish string) (*github.RepositoryRelease, error) {
	// Set the target commitish (SHA) for the release
	release.TargetCommitish = &targetCommitish
	
	createdRelease, _, err := c.Repositories.CreateRelease(c.ctx, repo.Owner, repo.Name, release)
	if err != nil {
		return nil, fmt.Errorf("failed to create release from SHA %s for %s: %w", targetCommitish, repo, err)
	}
	return createdRelease, nil
}

func (c *Client) GetPullRequestComments(repo *Repository, number int) ([]*github.IssueComment, error) {
	opts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}

	var allComments []*github.IssueComment
	
	for {
		comments, resp, err := c.Issues.ListComments(c.ctx, repo.Owner, repo.Name, number, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get comments for PR #%d in %s: %w", number, repo, err)
		}
		
		allComments = append(allComments, comments...)
		
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allComments, nil
}