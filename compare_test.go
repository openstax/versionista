package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/google/go-github/v28/github"
)

// fakeCompare serves a compare of `total` commits the way GitHub does: an
// unpaginated request gets at most 250, oldest first; a paginated one gets
// per_page commits per page with a Link header to the next page. Commit i is
// the squash merge of PR #i.
func fakeCompare(t *testing.T, total, served int) *Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/compare/v1...v2", func(w http.ResponseWriter, r *http.Request) {
		perPage, page := 250, 1
		if v := r.URL.Query().Get("per_page"); v != "" {
			perPage, _ = strconv.Atoi(v)
		}
		if v := r.URL.Query().Get("page"); v != "" {
			page, _ = strconv.Atoi(v)
		}
		start := (page - 1) * perPage
		end := start + perPage
		if end > served {
			end = served
		}
		if r.URL.Query().Get("page") == "" && end > 250 {
			end = 250
		}
		var commits []map[string]interface{}
		for i := start + 1; i <= end; i++ {
			commits = append(commits, map[string]interface{}{
				"sha":    fmt.Sprintf("sha%d", i),
				"commit": map[string]string{"message": fmt.Sprintf("Change %d (#%d)", i, i)},
			})
		}
		if end < served && r.URL.Query().Get("page") != "" {
			next := fmt.Sprintf("http://%s%s?per_page=%d&page=%d", r.Host, r.URL.Path, perPage, page+1)
			w.Header().Set("Link", fmt.Sprintf(`<%s>; rel="next"`, next))
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"total_commits": total, "commits": commits})
	})
	mux.HandleFunc("/repos/o/r/pulls/", func(w http.ResponseWriter, r *http.Request) {
		n, _ := strconv.Atoi(r.URL.Path[len("/repos/o/r/pulls/"):])
		json.NewEncoder(w).Encode(map[string]interface{}{"number": n, "title": fmt.Sprintf("PR %d", n)})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	gh := github.NewClient(nil)
	base, _ := url.Parse(server.URL + "/")
	gh.BaseURL = base
	return &Client{Client: gh, ctx: context.Background()}
}

func TestCompareCommitsReadsEveryPage(t *testing.T) {
	client := fakeCompare(t, 378, 378)
	comparison, err := client.CompareCommits(&Repository{Owner: "o", Name: "r"}, "v1", "v2")
	if err != nil {
		t.Fatal(err)
	}
	if got := len(comparison.Commits); got != 378 {
		t.Fatalf("got %d commits, want 378", got)
	}
	for i, c := range comparison.Commits {
		if want := fmt.Sprintf("sha%d", i+1); c.GetSHA() != want {
			t.Fatalf("commit %d is %s, want %s: pages are out of order", i, c.GetSHA(), want)
		}
	}
}

func TestCompareCommitsFailsWhenCommitsAreMissing(t *testing.T) {
	client := fakeCompare(t, 378, 300)
	if _, err := client.CompareCommits(&Repository{Owner: "o", Name: "r"}, "v1", "v2"); err == nil {
		t.Fatal("a compare short of total_commits must fail, not return a partial changelog")
	}
}

// The release that found this: v1.6.9 had 378 commits and its notes listed only
// the PRs in the first 250.
func TestChangelogKeepsPRsPastTheFirst250Commits(t *testing.T) {
	client := fakeCompare(t, 378, 378)
	manager := NewManager(client, NewLogger(), nil, "", true)
	repo := &ReleaseRepository{Repository: &Repository{Owner: "o", Name: "r"}}
	entries, err := manager.GenerateChangelogBetween(context.Background(), repo, "v1", "v2")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 378 {
		t.Fatalf("got %d entries, want 378", len(entries))
	}
	if last := entries[len(entries)-1].Number; last != 378 {
		t.Fatalf("last entry is #%d, want #378", last)
	}
}
