package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GenerateAssets runs the repo's configured generate-assets command using the
// default shell, passing the chosen version as a single argument ($1). Before
// running, it verifies the git working tree at dir is clean and checks out sha
// (the ref being released); once the command finishes it restores the branch or
// commit that was checked out beforehand. The version is passed to the command
// as an argument ($1), so a bare script path like ./release.sh receives it
// directly. The command runs with its working
// directory set to dir (the repo's configured `path`); if dir is empty it falls
// back to the directory versionista was invoked from. On success, stdout is
// parsed as a newline-separated list of file paths to upload alongside the
// release.
func GenerateAssets(command, version, dir, sha string) (paths []string, err error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to determine working directory: %w", err)
		}
		dir = cwd
	}

	if info, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("generate-assets path %q is not accessible: %w", dir, err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("generate-assets path %q is not a directory", dir)
	}

	original, err := prepareRepo(dir, sha)
	if err != nil {
		return nil, err
	}
	// Restore whatever was checked out before we started, regardless of outcome.
	defer func() {
		if restoreErr := restoreRepo(dir, original); restoreErr != nil && err == nil {
			err = restoreErr
		}
	}()

	// The -c script forwards its positional args to the command via "$@", so the
	// version reaches the command as $1. The name after the script string becomes
	// $0 (a conventional label), and version becomes $1.
	cmd := exec.Command(shell, "-c", command+` "$@"`, "generate-assets", version)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("generate-assets command failed: %s %q %q: %w\noutput:\n%s",
			shell, command, version, err, combinedOutput(stdout.String(), stderr.String()))
	}

	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Resolve relative paths against dir so the command can emit paths
		// relative to its own working directory.
		if !filepath.IsAbs(line) {
			line = filepath.Join(dir, line)
		}
		paths = append(paths, line)
	}

	return paths, nil
}

// prepareRepo verifies the git working tree at dir is clean, records the branch
// or commit currently checked out, then checks out the ref (sha) being released
// so generate-assets builds from the exact code that will be published. It
// returns the original ref so the caller can restore it afterwards.
//
// When ref is a branch name (the common case — the release config points at a
// branch like "main"), prepareRepo fetches the remote first and checks out the
// remote-tracking tip (origin/<branch>) rather than the possibly-stale LOCAL
// branch, so a release always builds what is actually published on the remote —
// not whatever the operator's local checkout happens to be. Fetch is best-effort:
// a repo with no remote (e.g. tests, air-gapped builds) falls back to checking
// out ref as-is, which also covers plain SHAs and tags.
func prepareRepo(dir, ref string) (string, error) {
	// Confirm dir is a git working tree.
	if _, err := runGit(dir, "rev-parse", "--is-inside-work-tree"); err != nil {
		return "", fmt.Errorf("%q is not a git repository: %w", dir, err)
	}

	// Refuse to proceed if there are uncommitted changes.
	status, err := runGit(dir, "status", "--porcelain")
	if err != nil {
		return "", fmt.Errorf("failed to check git status in %q: %w", dir, err)
	}
	if strings.TrimSpace(status) != "" {
		return "", fmt.Errorf("git working tree in %q is not clean; commit, stash, or discard changes before releasing:\n%s", dir, status)
	}

	original, err := currentRef(dir)
	if err != nil {
		return "", err
	}

	// Resolve the checkout target: prefer the up-to-date remote tip for a branch.
	target := resolveCheckoutTarget(dir, ref)
	if _, err := runGit(dir, "checkout", target); err != nil {
		return "", fmt.Errorf("failed to checkout %q in %q: %w", target, dir, err)
	}

	return original, nil
}

// resolveCheckoutTarget decides what prepareRepo should actually check out for
// the requested ref. If ref names a remote branch, it fetches that remote and
// returns the remote-tracking ref (e.g. "origin/main"), so the build reflects the
// published tip rather than a stale local branch. For SHAs, tags, or repos with
// no matching remote branch, it returns ref unchanged. All remote probing is
// best-effort — any failure falls back to ref so local-only repos still work.
func resolveCheckoutTarget(dir, ref string) string {
	remote := defaultRemote(dir)
	if remote == "" {
		return ref
	}
	// Only rewrite branch names; leave SHAs/tags alone. A ref is treated as a
	// branch when the remote advertises it (ls-remote --heads matches a line).
	heads, err := runGit(dir, "ls-remote", "--heads", remote, ref)
	if err != nil || strings.TrimSpace(heads) == "" {
		return ref
	}
	// Fetch just that branch so origin/<ref> is current, then target it.
	if _, err := runGit(dir, "fetch", remote, ref); err != nil {
		return ref // fetch failed — fall back to the local ref rather than abort.
	}
	return remote + "/" + ref
}

// defaultRemote returns the remote to fetch from — "origin" when present, else
// the first configured remote, or "" when the repo has none (local-only).
func defaultRemote(dir string) string {
	out, err := runGit(dir, "remote")
	if err != nil {
		return ""
	}
	remotes := strings.Fields(out)
	for _, r := range remotes {
		if r == "origin" {
			return "origin"
		}
	}
	if len(remotes) > 0 {
		return remotes[0]
	}
	return ""
}

// currentRef returns the branch name currently checked out at dir, or the
// commit SHA when in a detached HEAD state.
func currentRef(dir string) (string, error) {
	if branch, err := runGit(dir, "symbolic-ref", "--short", "-q", "HEAD"); err == nil {
		if b := strings.TrimSpace(branch); b != "" {
			return b, nil
		}
	}
	// Detached HEAD (or symbolic-ref returned nothing) — fall back to the SHA.
	sha, err := runGit(dir, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to determine current git ref in %q: %w", dir, err)
	}
	return strings.TrimSpace(sha), nil
}

// restoreRepo checks out ref, returning the working tree to the branch or commit
// that was active before prepareRepo ran.
func restoreRepo(dir, ref string) error {
	if ref == "" {
		return nil
	}
	if _, err := runGit(dir, "checkout", ref); err != nil {
		return fmt.Errorf("failed to restore %q in %q: %w", ref, dir, err)
	}
	return nil
}

// runGit runs a git subcommand in dir and returns its combined trimmed output.
// On failure the error includes git's stderr for context.
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s failed: %w\n%s", strings.Join(args, " "), err, combinedOutput(stdout.String(), stderr.String()))
	}
	return stdout.String(), nil
}

// combinedOutput joins captured stdout and stderr for inclusion in error
// messages, trimming each and skipping empty streams.
func combinedOutput(stdout, stderr string) string {
	var parts []string
	if s := strings.TrimSpace(stdout); s != "" {
		parts = append(parts, s)
	}
	if s := strings.TrimSpace(stderr); s != "" {
		parts = append(parts, s)
	}
	if len(parts) == 0 {
		return "(no output)"
	}
	return strings.Join(parts, "\n")
}
