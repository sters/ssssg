package ssssg

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupProject(t *testing.T, siteYAML string) string {
	t.Helper()

	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "site.yaml"), []byte(siteYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	tmplDir := filepath.Join(dir, "templates")
	if err := os.MkdirAll(tmplDir, 0o755); err != nil {
		t.Fatal(err)
	}

	staticDir := filepath.Join(dir, "static")
	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		t.Fatal(err)
	}

	return dir
}

func TestBuild_Basic(t *testing.T) {
	t.Parallel()

	yaml := `
global:
  data:
    site_name: "Test"

pages:
  - template: "index.html"
    output: "index.html"
    data:
      title: "Home"
      greeting: "Hello World"
`

	dir := setupProject(t, yaml)

	// Create template
	tmpl := `<html><body><h1>{{ .Page.greeting }}</h1><p>{{ .Global.site_name }}</p></body></html>`
	if err := os.WriteFile(filepath.Join(dir, "templates", "index.html"), []byte(tmpl), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create static file
	if err := os.WriteFile(filepath.Join(dir, "static", "style.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Build(t.Context(), BuildOptions{
		ConfigPath: filepath.Join(dir, "site.yaml"),
		Timeout:    10 * time.Second,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Check output HTML
	content, err := os.ReadFile(filepath.Join(dir, "public", "index.html"))
	if err != nil {
		t.Fatal("output file not created")
	}

	html := string(content)
	if !strings.Contains(html, "<h1>Hello World</h1>") {
		t.Errorf("missing greeting in output:\n%s", html)
	}

	if !strings.Contains(html, "<p>Test</p>") {
		t.Errorf("missing site_name in output:\n%s", html)
	}

	// Check static file copied
	css, err := os.ReadFile(filepath.Join(dir, "public", "style.css"))
	if err != nil {
		t.Fatal("static file not copied")
	}

	if string(css) != "body{}" {
		t.Errorf("static file content = %q", string(css))
	}
}

func TestBuild_WithLayout(t *testing.T) {
	t.Parallel()

	yaml := `
global:
  layout: "_layout.html"
  data:
    site_name: "My Site"

pages:
  - template: "index.html"
    output: "index.html"
    data:
      title: "Home"
      greeting: "Welcome"
`

	dir := setupProject(t, yaml)

	layout := `<!DOCTYPE html><html><head><title>{{ .Page.title }}</title></head><body>{{ block "content" . }}{{ end }}</body></html>`
	if err := os.WriteFile(filepath.Join(dir, "templates", "_layout.html"), []byte(layout), 0o644); err != nil {
		t.Fatal(err)
	}

	page := `{{ define "content" }}<h1>{{ .Page.greeting }}</h1>{{ end }}`
	if err := os.WriteFile(filepath.Join(dir, "templates", "index.html"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Build(t.Context(), BuildOptions{
		ConfigPath: filepath.Join(dir, "site.yaml"),
		Timeout:    10 * time.Second,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "public", "index.html"))
	if err != nil {
		t.Fatal(err)
	}

	html := string(content)
	if !strings.Contains(html, "<title>Home</title>") {
		t.Errorf("missing title:\n%s", html)
	}

	if !strings.Contains(html, "<h1>Welcome</h1>") {
		t.Errorf("missing content:\n%s", html)
	}
}

func TestBuild_WithFetch(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("fetched content"))
	}))
	defer srv.Close()

	yaml := `
global:
  data:
    site_name: "Test"

pages:
  - template: "index.html"
    output: "index.html"
    data:
      title: "Home"
    fetch:
      remote: "` + srv.URL + `"
`

	dir := setupProject(t, yaml)

	tmpl := `<html><body>{{ .Page.remote }}</body></html>`
	if err := os.WriteFile(filepath.Join(dir, "templates", "index.html"), []byte(tmpl), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Build(t.Context(), BuildOptions{
		ConfigPath: filepath.Join(dir, "site.yaml"),
		Timeout:    10 * time.Second,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "public", "index.html"))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "fetched content") {
		t.Errorf("missing fetched content:\n%s", string(content))
	}
}

func TestBuild_WithLocalFetch(t *testing.T) {
	t.Parallel()

	yaml := `
global:
  fetch:
    css: "static/style.css"

pages:
  - template: "index.html"
    output: "index.html"
    data:
      title: "Home"
`

	dir := setupProject(t, yaml)

	if err := os.WriteFile(filepath.Join(dir, "static", "style.css"), []byte("body{margin:0}"), 0o644); err != nil {
		t.Fatal(err)
	}

	tmpl := `<style>{{ .Global.css | rawCSS }}</style>`
	if err := os.WriteFile(filepath.Join(dir, "templates", "index.html"), []byte(tmpl), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Build(t.Context(), BuildOptions{
		ConfigPath: filepath.Join(dir, "site.yaml"),
		Timeout:    10 * time.Second,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "public", "index.html"))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "body{margin:0}") {
		t.Errorf("missing local fetch content:\n%s", string(content))
	}
}

func TestBuild_Clean(t *testing.T) {
	t.Parallel()

	yaml := `
global:
  data:
    site_name: "Test"

pages:
  - template: "index.html"
    output: "index.html"
    data:
      title: "Home"
`

	dir := setupProject(t, yaml)

	tmpl := `<html><body>{{ .Page.title }}</body></html>`
	if err := os.WriteFile(filepath.Join(dir, "templates", "index.html"), []byte(tmpl), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create stale file in output directory
	publicDir := filepath.Join(dir, "public")
	if err := os.MkdirAll(publicDir, 0o755); err != nil {
		t.Fatal(err)
	}

	staleFile := filepath.Join(publicDir, "stale.html")
	if err := os.WriteFile(staleFile, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Build(t.Context(), BuildOptions{
		ConfigPath: filepath.Join(dir, "site.yaml"),
		Timeout:    10 * time.Second,
		Clean:      true,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Stale file should be removed
	if _, err := os.Stat(staleFile); err == nil {
		t.Error("stale file should have been removed by clean")
	}

	// New file should exist
	if _, err := os.Stat(filepath.Join(publicDir, "index.html")); err != nil {
		t.Error("index.html should exist after build")
	}
}

func TestBuild_WithStaticPipelines(t *testing.T) {
	t.Parallel()

	yaml := `
global:
  data:
    site_name: "Test"

pages:
  - template: "index.html"
    output: "index.html"
    data:
      title: "Home"

static:
  pipelines:
    - match: "*.txt"
      commands:
        - "cp {{.Src}} {{.Dest}}"
        - "sh -c 'echo PROCESSED >> {{.Dest}}'"
`

	dir := setupProject(t, yaml)

	tmpl := `<html><body>{{ .Page.title }}</body></html>`
	if err := os.WriteFile(filepath.Join(dir, "templates", "index.html"), []byte(tmpl), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create static files: one matching pipeline, one not
	if err := os.WriteFile(filepath.Join(dir, "static", "data.txt"), []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "static", "style.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Build(t.Context(), BuildOptions{
		ConfigPath: filepath.Join(dir, "site.yaml"),
		Timeout:    10 * time.Second,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Check pipeline-processed file
	txtContent, err := os.ReadFile(filepath.Join(dir, "public", "data.txt"))
	if err != nil {
		t.Fatal("data.txt not created")
	}

	if !strings.Contains(string(txtContent), "original") {
		t.Errorf("data.txt missing original content: %q", string(txtContent))
	}

	if !strings.Contains(string(txtContent), "PROCESSED") {
		t.Errorf("data.txt missing PROCESSED marker: %q", string(txtContent))
	}

	// Check unmatched file was copied normally
	cssContent, err := os.ReadFile(filepath.Join(dir, "public", "style.css"))
	if err != nil {
		t.Fatal("style.css not copied")
	}

	if string(cssContent) != "body{}" {
		t.Errorf("style.css content = %q, want %q", string(cssContent), "body{}")
	}
}
