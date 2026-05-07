package main

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
)

type reviewSection struct {
	RepoFullName string
	CurrentVer   string
	Markdown     string
}

func renderReviewHTML(projectName string, sections []reviewSection) (string, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(goldmarkhtml.WithUnsafe()),
	)

	var body strings.Builder
	body.WriteString(`<nav class="toc"><h2>Repositories</h2><ul>`)
	for _, s := range sections {
		body.WriteString(fmt.Sprintf(
			`<li><a href="#%s">%s</a> <span class="ver">%s</span></li>`,
			anchorID(s.RepoFullName), htmlEscape(s.RepoFullName), htmlEscape(s.CurrentVer),
		))
	}
	body.WriteString(`</ul></nav>`)

	for _, s := range sections {
		var rendered bytes.Buffer
		if err := md.Convert([]byte(s.Markdown), &rendered); err != nil {
			return "", fmt.Errorf("render markdown for %s: %w", s.RepoFullName, err)
		}
		body.WriteString(fmt.Sprintf(
			`<section id="%s"><h2>%s <span class="ver">current %s</span></h2>%s</section>`,
			anchorID(s.RepoFullName), htmlEscape(s.RepoFullName), htmlEscape(s.CurrentVer), rendered.String(),
		))
	}

	title := htmlEscape(projectName)
	page := fmt.Sprintf(htmlShell, title, title, time.Now().Format(time.RFC1123), body.String())
	return page, nil
}

func writeReviewHTML(html string) (string, error) {
	f, err := os.CreateTemp("", "versionista-review-*.html")
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(html); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func openInBrowser(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", abs).Run()
	case "linux":
		return exec.Command("xdg-open", abs).Run()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", abs).Run()
	default:
		return fmt.Errorf("don't know how to open browser on %s", runtime.GOOS)
	}
}

func anchorID(repoFullName string) string {
	return strings.NewReplacer("/", "-", " ", "-").Replace(repoFullName)
}

func htmlEscape(s string) string { return html.EscapeString(s) }

const htmlShell = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Versionista review — %s</title>
<style>
  :root { color-scheme: light dark; }
  body { font: 14px/1.5 -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif;
         margin: 0; padding: 2rem; max-width: 1100px; margin-inline: auto; }
  h1 { font-size: 1.6rem; margin: 0 0 .25rem; }
  h2 { font-size: 1.2rem; margin: 2rem 0 .75rem; padding-bottom: .25rem;
       border-bottom: 1px solid color-mix(in srgb, currentColor 15%%, transparent); }
  .meta { color: color-mix(in srgb, currentColor 60%%, transparent); margin-bottom: 1.5rem; }
  .ver { font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
         font-size: .85em; color: color-mix(in srgb, currentColor 60%%, transparent); font-weight: normal; }
  nav.toc { padding: 1rem 1.25rem; border: 1px solid color-mix(in srgb, currentColor 15%%, transparent);
            border-radius: 8px; margin-bottom: 2rem; }
  nav.toc h2 { margin: 0 0 .5rem; padding: 0; border: 0; font-size: 1rem; }
  nav.toc ul { margin: 0; padding-left: 1.25rem; }
  nav.toc li { margin: .15rem 0; }
  table { border-collapse: collapse; width: 100%%; margin: .5rem 0 1rem;
          font-size: .92em; }
  th, td { border: 1px solid color-mix(in srgb, currentColor 15%%, transparent);
           padding: .4rem .6rem; text-align: left; vertical-align: top; }
  th { background: color-mix(in srgb, currentColor 6%%, transparent); }
  tr:nth-child(even) td { background: color-mix(in srgb, currentColor 3%%, transparent); }
  details summary { cursor: pointer; }
  details[open] summary { margin-bottom: .35rem; }
  a { color: #0366d6; }
  @media (prefers-color-scheme: dark) { a { color: #79b8ff; } }
  hr { border: 0; border-top: 1px solid color-mix(in srgb, currentColor 15%%, transparent); margin: 1.5rem 0; }
</style>
</head>
<body>
<h1>Versionista review — %s</h1>
<p class="meta">Generated %s</p>
%s
</body>
</html>
`
