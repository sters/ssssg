package ssssg

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	yaml := `
global:
  layout: "_layout.html"
  data:
    site_name: "Test Site"
  fetch:
    reset_css: "https://example.com/reset.css"

pages:
  - template: "index.html"
    output: "index.html"
    data:
      title: "Home"
  - template: "about.html"
    output: "about/index.html"
    layout: "_other_layout.html"
    data:
      title: "About"
    fetch:
      bio: "data/bio.txt"
`

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "site.yaml")
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Global.Layout != "_layout.html" {
		t.Errorf("global.layout = %q, want %q", cfg.Global.Layout, "_layout.html")
	}

	if cfg.Global.Data["site_name"] != "Test Site" {
		t.Errorf("global.data.site_name = %v, want %q", cfg.Global.Data["site_name"], "Test Site")
	}

	if cfg.Global.Fetch["reset_css"] != "https://example.com/reset.css" {
		t.Errorf("global.fetch.reset_css = %v", cfg.Global.Fetch["reset_css"])
	}

	if len(cfg.Pages) != 2 {
		t.Fatalf("len(pages) = %d, want 2", len(cfg.Pages))
	}

	p0 := cfg.Pages[0]
	if p0.Template != "index.html" || p0.Output != "index.html" {
		t.Errorf("pages[0] template=%q output=%q", p0.Template, p0.Output)
	}

	if p0.Data["title"] != "Home" {
		t.Errorf("pages[0].data.title = %v", p0.Data["title"])
	}

	p1 := cfg.Pages[1]
	if p1.Layout != "_other_layout.html" {
		t.Errorf("pages[1].layout = %q", p1.Layout)
	}

	if p1.Fetch["bio"] != "data/bio.txt" {
		t.Errorf("pages[1].fetch.bio = %v", p1.Fetch["bio"])
	}
}

func TestLoadConfig_MissingTemplate(t *testing.T) {
	t.Parallel()

	yaml := `
pages:
  - output: "index.html"
`

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "site.yaml")
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(cfgPath)
	if err == nil {
		t.Fatal("expected error for missing template")
	}
}

func TestLoadConfig_MissingOutput(t *testing.T) {
	t.Parallel()

	yaml := `
pages:
  - template: "index.html"
`

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "site.yaml")
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(cfgPath)
	if err == nil {
		t.Fatal("expected error for missing output")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	t.Parallel()

	_, err := LoadConfig("/nonexistent/site.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoadConfig_PathTraversal(t *testing.T) {
	t.Parallel()

	yaml := `
pages:
  - template: "index.html"
    output: "../../etc/passwd"
`

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "site.yaml")
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(cfgPath)
	if err == nil {
		t.Fatal("expected error for path traversal")
	}

	if !errors.Is(err, errOutputPathTraversal) {
		t.Errorf("expected errOutputPathTraversal, got: %v", err)
	}
}

func TestLoadConfig_WithStaticPipelines(t *testing.T) {
	t.Parallel()

	yaml := `
pages:
  - template: "index.html"
    output: "index.html"

static:
  pipelines:
    - match: "*.jpg"
      commands:
        - "cp {{.Src}} {{.Dest}}"
        - "echo done"
    - match: "images/*.png"
      commands:
        - "cp {{.Src}} {{.Dest}}"
`

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "site.yaml")
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if len(cfg.Static.Pipelines) != 2 {
		t.Fatalf("len(pipelines) = %d, want 2", len(cfg.Static.Pipelines))
	}

	if cfg.Static.Pipelines[0].Match != "*.jpg" {
		t.Errorf("pipelines[0].match = %q, want %q", cfg.Static.Pipelines[0].Match, "*.jpg")
	}

	if len(cfg.Static.Pipelines[0].Commands) != 2 {
		t.Errorf("pipelines[0].commands len = %d, want 2", len(cfg.Static.Pipelines[0].Commands))
	}

	if cfg.Static.Pipelines[1].Match != "images/*.png" {
		t.Errorf("pipelines[1].match = %q, want %q", cfg.Static.Pipelines[1].Match, "images/*.png")
	}
}

func TestLoadConfig_PipelineMissingMatch(t *testing.T) {
	t.Parallel()

	yaml := `
pages:
  - template: "index.html"
    output: "index.html"

static:
  pipelines:
    - commands:
        - "echo hello"
`

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "site.yaml")
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(cfgPath)
	if err == nil {
		t.Fatal("expected error for missing match")
	}

	if !errors.Is(err, errPipelineMatchEmpty) {
		t.Errorf("expected errPipelineMatchEmpty, got: %v", err)
	}
}

func TestLoadConfig_PipelineNoCommands(t *testing.T) {
	t.Parallel()

	yaml := `
pages:
  - template: "index.html"
    output: "index.html"

static:
  pipelines:
    - match: "*.jpg"
`

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "site.yaml")
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(cfgPath)
	if err == nil {
		t.Fatal("expected error for no commands")
	}

	if !errors.Is(err, errPipelineNoCommands) {
		t.Errorf("expected errPipelineNoCommands, got: %v", err)
	}
}

func TestLoadConfig_PipelineInvalidPattern(t *testing.T) {
	t.Parallel()

	yaml := `
pages:
  - template: "index.html"
    output: "index.html"

static:
  pipelines:
    - match: "[invalid"
      commands:
        - "echo hello"
`

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "site.yaml")
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid pattern")
	}

	if !errors.Is(err, errPipelineInvalidMatch) {
		t.Errorf("expected errPipelineInvalidMatch, got: %v", err)
	}
}

func TestLoadConfig_AbsoluteOutputPath(t *testing.T) {
	t.Parallel()

	yaml := `
pages:
  - template: "index.html"
    output: "/tmp/evil.html"
`

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "site.yaml")
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(cfgPath)
	if err == nil {
		t.Fatal("expected error for absolute output path")
	}

	if !errors.Is(err, errOutputPathTraversal) {
		t.Errorf("expected errOutputPathTraversal, got: %v", err)
	}
}
