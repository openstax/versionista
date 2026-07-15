package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initTestRepo creates a git repository in a temp dir with a single commit and
// returns the directory and the commit SHA. It configures a local identity so
// commits succeed regardless of the host's git config.
func initTestRepo(t *testing.T) (dir, sha string) {
	t.Helper()
	dir = t.TempDir()

	run := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}

	run("init", "-q")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	run("config", "commit.gpgsign", "false")
	run("config", "tag.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "initial")
	sha = run("rev-parse", "HEAD")
	return dir, sha
}

func TestGenerateAssetsReturnsPaths(t *testing.T) {
	dir, sha := initTestRepo(t)
	// The command receives the version as $1 and echoes two paths built from it.
	paths, err := GenerateAssets("printf 'dist/app-%s.tar.gz\\ndist/app-%s.zip\\n' \"$1\" \"$1\" #", "1.2.3", dir, sha)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expected := []string{
		filepath.Join(dir, "dist/app-1.2.3.tar.gz"),
		filepath.Join(dir, "dist/app-1.2.3.zip"),
	}
	if len(paths) != len(expected) {
		t.Fatalf("Expected %d paths, got %d: %v", len(expected), len(paths), paths)
	}
	for i, want := range expected {
		if paths[i] != want {
			t.Errorf("path %d: expected %q, got %q", i, want, paths[i])
		}
	}
}

func TestGenerateAssetsIgnoresBlankLines(t *testing.T) {
	dir, sha := initTestRepo(t)
	paths, err := GenerateAssets("printf 'one\\n\\n  two  \\n'", "1.0.0", dir, sha)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	expected := []string{filepath.Join(dir, "one"), filepath.Join(dir, "two")}
	if len(paths) != len(expected) {
		t.Fatalf("Expected %d paths, got %d: %v", len(expected), len(paths), paths)
	}
	for i, want := range expected {
		if paths[i] != want {
			t.Errorf("path %d: expected %q, got %q", i, want, paths[i])
		}
	}
}

func TestGenerateAssetsAbsolutePathsKept(t *testing.T) {
	dir, sha := initTestRepo(t)
	paths, err := GenerateAssets("echo /tmp/absolute-$1.zip #", "1.0.0", dir, sha)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(paths) != 1 || paths[0] != "/tmp/absolute-1.0.0.zip" {
		t.Errorf("Expected [/tmp/absolute-1.0.0.zip], got: %v", paths)
	}
}

func TestGenerateAssetsFailureReturnsError(t *testing.T) {
	dir, sha := initTestRepo(t)
	if _, err := GenerateAssets("exit 1", "1.0.0", dir, sha); err == nil {
		t.Fatal("Expected an error when the command exits non-zero")
	}
}

func TestGenerateAssetsFailureIncludesCommandAndOutput(t *testing.T) {
	dir, sha := initTestRepo(t)
	_, err := GenerateAssets("echo boom-out; echo boom-err >&2; exit 3", "1.0.0", dir, sha)
	if err == nil {
		t.Fatal("Expected an error when the command exits non-zero")
	}
	msg := err.Error()
	for _, want := range []string{"echo boom-out", "boom-out", "boom-err"} {
		if !strings.Contains(msg, want) {
			t.Errorf("Expected error to contain %q, got: %s", want, msg)
		}
	}
}

func TestGenerateAssetsRunsInConfiguredDir(t *testing.T) {
	dir, sha := initTestRepo(t)
	// `ls README` only succeeds if the command runs in dir. The trailing `#`
	// comments out the appended version argument.
	paths, err := GenerateAssets("ls README >/dev/null && echo README #", "1.0.0", dir, sha)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(paths) != 1 || paths[0] != filepath.Join(dir, "README") {
		t.Errorf("Expected [%s], got: %v", filepath.Join(dir, "README"), paths)
	}
}

// TestGenerateAssetsForwardsArgsToScript mirrors the real-world config where
// generate-assets is a script path rather than an inline command. The script
// requires exactly one argument, so this fails unless the version is forwarded
// to the script as $1 (not left as a positional arg of the shell's -c string).
func TestGenerateAssetsForwardsArgsToScript(t *testing.T) {
	dir, sha := initTestRepo(t)
	script := filepath.Join(dir, "release.sh")
	body := "#!/bin/sh\nif [ \"$#\" -ne 1 ]; then echo 'usage: release.sh <version>' >&2; exit 1; fi\necho \"dist/app-$1.zip\"\n"
	if err := os.WriteFile(script, []byte(body), 0755); err != nil {
		t.Fatal(err)
	}
	// Commit the script so the working tree stays clean.
	for _, args := range [][]string{{"add", "."}, {"commit", "-qm", "add script"}} {
		g := exec.Command("git", args...)
		g.Dir = dir
		if out, err := g.CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	newSHA := exec.Command("git", "rev-parse", "HEAD")
	newSHA.Dir = dir
	shaOut, err := newSHA.Output()
	if err != nil {
		t.Fatal(err)
	}
	sha = strings.TrimSpace(string(shaOut))

	paths, err := GenerateAssets("./release.sh", "v0.0.1", dir, sha)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	want := filepath.Join(dir, "dist/app-v0.0.1.zip")
	if len(paths) != 1 || paths[0] != want {
		t.Errorf("Expected [%s], got: %v", want, paths)
	}
}

func TestGenerateAssetsRejectsDirtyTree(t *testing.T) {
	dir, sha := initTestRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := GenerateAssets("echo out", "1.0.0", dir, sha)
	if err == nil {
		t.Fatal("Expected an error when the working tree is dirty")
	}
	if !strings.Contains(err.Error(), "not clean") {
		t.Errorf("Expected a 'not clean' error, got: %v", err)
	}
}

func TestGenerateAssetsRejectsNonGitDir(t *testing.T) {
	dir := t.TempDir()
	if _, err := GenerateAssets("echo out", "1.0.0", dir, "HEAD"); err == nil {
		t.Fatal("Expected an error when the directory is not a git repository")
	}
}

func TestGenerateAssetsRestoresOriginalBranch(t *testing.T) {
	dir, firstSHA := initTestRepo(t)

	git := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}

	branch := git("rev-parse", "--abbrev-ref", "HEAD")

	// Add a second commit so there's a distinct SHA to check out.
	if err := os.WriteFile(filepath.Join(dir, "second.txt"), []byte("2"), 0644); err != nil {
		t.Fatal(err)
	}
	git("add", ".")
	git("commit", "-q", "-m", "second")

	// Release the first (older) commit; afterwards we should be back on branch.
	if _, err := GenerateAssets("echo out", "1.0.0", dir, firstSHA); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if got := git("rev-parse", "--abbrev-ref", "HEAD"); got != branch {
		t.Errorf("Expected to be restored to branch %q, got %q", branch, got)
	}
}

// gitIn runs a git command in dir, failing the test on error.
func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s (in %s) failed: %v\n%s", strings.Join(args, " "), dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

// TestGenerateAssetsBuildsRemoteBranchTip is the regression test for the stale
// local-branch bug: when the release ref is a BRANCH NAME and the remote is
// ahead of the local branch, the build must run against the REMOTE tip (what is
// actually published), not the operator's stale local checkout.
func TestGenerateAssetsBuildsRemoteBranchTip(t *testing.T) {
	// A bare repo acts as the remote, seeded with one commit via a scratch clone.
	remote := t.TempDir()
	gitIn(t, remote, "init", "-q", "--bare")

	seed := t.TempDir()
	gitIn(t, seed, "clone", "-q", remote, ".")
	gitIn(t, seed, "config", "user.email", "t@example.com")
	gitIn(t, seed, "config", "user.name", "T")
	gitIn(t, seed, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(seed, "marker.txt"), []byte("v1"), 0644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, seed, "add", ".")
	gitIn(t, seed, "commit", "-q", "-m", "v1")
	branch := gitIn(t, seed, "rev-parse", "--abbrev-ref", "HEAD")
	gitIn(t, seed, "push", "-q", "origin", branch)

	// The operator's working copy: clone the remote at v1 (its local branch is now
	// one commit BEHIND once we push v2 to the remote below).
	work := t.TempDir()
	gitIn(t, work, "clone", "-q", remote, ".")

	// Advance the REMOTE to v2 via the seed clone. work's local branch still at v1.
	if err := os.WriteFile(filepath.Join(seed, "marker.txt"), []byte("v2"), 0644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, seed, "add", ".")
	gitIn(t, seed, "commit", "-q", "-m", "v2")
	gitIn(t, seed, "push", "-q", "origin", branch)

	// Sanity: work's local branch is behind (still reads v1).
	if got := gitIn(t, work, "show", "HEAD:marker.txt"); got != "v1" {
		t.Fatalf("precondition: work should be at v1, got %q", got)
	}

	// Release by BRANCH NAME. The generate-assets command writes marker.txt's
	// content to an output file so we can prove which commit was built.
	outFile := filepath.Join(work, "built-content.txt")
	cmd := "cp marker.txt " + outFile + " && echo " + outFile
	if _, err := GenerateAssets(cmd, "1.0.0", work, branch); err != nil {
		t.Fatalf("GenerateAssets returned error: %v", err)
	}

	built, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading built content: %v", err)
	}
	if strings.TrimSpace(string(built)) != "v2" {
		t.Errorf("build ran against %q, want the remote tip %q — stale local branch was used", strings.TrimSpace(string(built)), "v2")
	}

	// And the working copy is restored to its original branch afterwards.
	if got := gitIn(t, work, "rev-parse", "--abbrev-ref", "HEAD"); got != branch {
		t.Errorf("expected restore to branch %q, got %q", branch, got)
	}
}

// TestGenerateAssetsExplicitSHANotRewritten confirms that when the ref is a
// commit SHA (not a branch), it is built as-is — the remote-branch resolution
// must not hijack an explicit SHA.
func TestGenerateAssetsExplicitSHANotRewritten(t *testing.T) {
	remote := t.TempDir()
	gitIn(t, remote, "init", "-q", "--bare")

	work := t.TempDir()
	gitIn(t, work, "clone", "-q", remote, ".")
	gitIn(t, work, "config", "user.email", "t@example.com")
	gitIn(t, work, "config", "user.name", "T")
	gitIn(t, work, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(work, "marker.txt"), []byte("first"), 0644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, work, "add", ".")
	gitIn(t, work, "commit", "-q", "-m", "first")
	firstSHA := gitIn(t, work, "rev-parse", "HEAD")
	branch := gitIn(t, work, "rev-parse", "--abbrev-ref", "HEAD")
	gitIn(t, work, "push", "-q", "origin", branch)

	// Second commit, pushed — so the branch tip differs from firstSHA.
	if err := os.WriteFile(filepath.Join(work, "marker.txt"), []byte("second"), 0644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, work, "add", ".")
	gitIn(t, work, "commit", "-q", "-m", "second")
	gitIn(t, work, "push", "-q", "origin", branch)

	outFile := filepath.Join(work, "built-content.txt")
	cmd := "cp marker.txt " + outFile + " && echo " + outFile
	if _, err := GenerateAssets(cmd, "1.0.0", work, firstSHA); err != nil {
		t.Fatalf("GenerateAssets returned error: %v", err)
	}

	built, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading built content: %v", err)
	}
	if strings.TrimSpace(string(built)) != "first" {
		t.Errorf("explicit SHA build ran against %q, want %q", strings.TrimSpace(string(built)), "first")
	}
}
