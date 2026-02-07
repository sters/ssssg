package ssssg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTemplateDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	tmplDir := filepath.Join(dir, "templates")
	if err := os.MkdirAll(tmplDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Layout
	layout := `<!DOCTYPE html>
<html>
<head><title>{{ .Page.title }} - {{ .Global.site_name }}</title></head>
<body>
{{ template "_header.html" . }}
{{ block "content" . }}{{ end }}
</body>
</html>`
	if err := os.WriteFile(filepath.Join(tmplDir, "_layout.html"), []byte(layout), 0o644); err != nil {
		t.Fatal(err)
	}

	// Header partial
	header := `<header><nav>{{ .Global.site_name }}</nav></header>`
	if err := os.WriteFile(filepath.Join(tmplDir, "_header.html"), []byte(header), 0o644); err != nil {
		t.Fatal(err)
	}

	// Page template
	page := `{{ define "content" }}<h1>{{ .Page.greeting }}</h1>{{ end }}`
	if err := os.WriteFile(filepath.Join(tmplDir, "index.html"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}

	// Standalone page (no layout)
	standalone := `<html><body><h1>{{ .Page.title }}</h1></body></html>`
	if err := os.WriteFile(filepath.Join(tmplDir, "standalone.html"), []byte(standalone), 0o644); err != nil {
		t.Fatal(err)
	}

	return dir
}

func TestRenderPage_WithLayout(t *testing.T) {
	t.Parallel()

	dir := setupTemplateDir(t)
	tmplDir := filepath.Join(dir, "templates")
	outputDir := filepath.Join(dir, "public")

	page := PageConfig{
		Template: "index.html",
		Output:   "index.html",
	}

	data := TemplateData{
		Global: map[string]any{"site_name": "Test Site"},
		Page:   map[string]any{"title": "Home", "greeting": "Hello!"},
	}

	err := RenderPage(tmplDir, page, "_layout.html", data, outputDir)
	if err != nil {
		t.Fatalf("RenderPage failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}

	html := string(content)
	if !strings.Contains(html, "<title>Home - Test Site</title>") {
		t.Errorf("missing title in output:\n%s", html)
	}

	if !strings.Contains(html, "<h1>Hello!</h1>") {
		t.Errorf("missing greeting in output:\n%s", html)
	}

	if !strings.Contains(html, "<header><nav>Test Site</nav></header>") {
		t.Errorf("missing header partial in output:\n%s", html)
	}
}

func TestRenderPage_WithoutLayout(t *testing.T) {
	t.Parallel()

	dir := setupTemplateDir(t)
	tmplDir := filepath.Join(dir, "templates")
	outputDir := filepath.Join(dir, "public")

	page := PageConfig{
		Template: "standalone.html",
		Output:   "standalone.html",
	}

	data := TemplateData{
		Global: map[string]any{},
		Page:   map[string]any{"title": "Standalone"},
	}

	err := RenderPage(tmplDir, page, "", data, outputDir)
	if err != nil {
		t.Fatalf("RenderPage failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "standalone.html"))
	if err != nil {
		t.Fatal(err)
	}

	html := string(content)
	if !strings.Contains(html, "<h1>Standalone</h1>") {
		t.Errorf("missing title in output:\n%s", html)
	}
}

func TestRenderPage_SubdirectoryOutput(t *testing.T) {
	t.Parallel()

	dir := setupTemplateDir(t)
	tmplDir := filepath.Join(dir, "templates")
	outputDir := filepath.Join(dir, "public")

	page := PageConfig{
		Template: "standalone.html",
		Output:   "about/index.html",
	}

	data := TemplateData{
		Global: map[string]any{},
		Page:   map[string]any{"title": "About"},
	}

	err := RenderPage(tmplDir, page, "", data, outputDir)
	if err != nil {
		t.Fatalf("RenderPage failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "about", "index.html"))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "<h1>About</h1>") {
		t.Errorf("missing title in output")
	}
}

func TestRenderPage_PageLayoutOverride(t *testing.T) {
	t.Parallel()

	dir := setupTemplateDir(t)
	tmplDir := filepath.Join(dir, "templates")
	outputDir := filepath.Join(dir, "public")

	// Create another layout
	otherLayout := `<html><body><div class="other">{{ block "content" . }}{{ end }}</div></body></html>`
	if err := os.WriteFile(filepath.Join(tmplDir, "_other_layout.html"), []byte(otherLayout), 0o644); err != nil {
		t.Fatal(err)
	}

	page := PageConfig{
		Template: "index.html",
		Output:   "index.html",
		Layout:   "_other_layout.html",
	}

	data := TemplateData{
		Global: map[string]any{"site_name": "Test"},
		Page:   map[string]any{"greeting": "Hi!"},
	}

	err := RenderPage(tmplDir, page, "_layout.html", data, outputDir)
	if err != nil {
		t.Fatalf("RenderPage failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}

	html := string(content)
	if !strings.Contains(html, `<div class="other">`) {
		t.Errorf("expected other layout in output:\n%s", html)
	}
}

func TestRenderPage_RawCSSFunction(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tmplDir := filepath.Join(dir, "templates")
	outputDir := filepath.Join(dir, "public")
	if err := os.MkdirAll(tmplDir, 0o755); err != nil {
		t.Fatal(err)
	}

	page := `<style>{{ .Page.css | rawCSS }}</style>`
	if err := os.WriteFile(filepath.Join(tmplDir, "raw.html"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := PageConfig{
		Template: "raw.html",
		Output:   "raw.html",
	}

	data := TemplateData{
		Global: map[string]any{},
		Page:   map[string]any{"css": "body { color: red; }"},
	}

	err := RenderPage(tmplDir, cfg, "", data, outputDir)
	if err != nil {
		t.Fatalf("RenderPage failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "raw.html"))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "body { color: red; }") {
		t.Errorf("raw content not preserved:\n%s", string(content))
	}
}

func TestRenderPage_RawHTMLFunction(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tmplDir := filepath.Join(dir, "templates")
	outputDir := filepath.Join(dir, "public")
	if err := os.MkdirAll(tmplDir, 0o755); err != nil {
		t.Fatal(err)
	}

	page := `<div>{{ .Page.content | raw }}</div>`
	if err := os.WriteFile(filepath.Join(tmplDir, "rawhtml.html"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := PageConfig{
		Template: "rawhtml.html",
		Output:   "rawhtml.html",
	}

	data := TemplateData{
		Global: map[string]any{},
		Page:   map[string]any{"content": "<strong>bold</strong>"},
	}

	err := RenderPage(tmplDir, cfg, "", data, outputDir)
	if err != nil {
		t.Fatalf("RenderPage failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "rawhtml.html"))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "<strong>bold</strong>") {
		t.Errorf("raw HTML not preserved:\n%s", string(content))
	}
}

func TestRenderPage_HTMLEscaping(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tmplDir := filepath.Join(dir, "templates")
	outputDir := filepath.Join(dir, "public")
	if err := os.MkdirAll(tmplDir, 0o755); err != nil {
		t.Fatal(err)
	}

	page := `<div>{{ .Page.content }}</div>`
	if err := os.WriteFile(filepath.Join(tmplDir, "escape.html"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := PageConfig{
		Template: "escape.html",
		Output:   "escape.html",
	}

	data := TemplateData{
		Global: map[string]any{},
		Page:   map[string]any{"content": "<script>alert('xss')</script>"},
	}

	err := RenderPage(tmplDir, cfg, "", data, outputDir)
	if err != nil {
		t.Fatalf("RenderPage failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "escape.html"))
	if err != nil {
		t.Fatal(err)
	}

	html := string(content)
	if strings.Contains(html, "<script>") {
		t.Errorf("script tag should be escaped:\n%s", html)
	}

	if !strings.Contains(html, "&lt;script&gt;") {
		t.Errorf("expected HTML-escaped content:\n%s", html)
	}
}

func TestCopyStatic(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	staticDir := filepath.Join(dir, "static")
	outputDir := filepath.Join(dir, "public")

	// Create static files
	if err := os.MkdirAll(filepath.Join(staticDir, "images"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(staticDir, "style.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(staticDir, "images", "logo.png"), []byte("PNG"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(staticDir, ".gitkeep"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	err := CopyStatic(staticDir, outputDir)
	if err != nil {
		t.Fatalf("CopyStatic failed: %v", err)
	}

	// Check style.css
	content, err := os.ReadFile(filepath.Join(outputDir, "style.css"))
	if err != nil {
		t.Fatal("style.css not copied")
	}

	if string(content) != "body{}" {
		t.Errorf("style.css content = %q", string(content))
	}

	// Check images/logo.png
	content, err = os.ReadFile(filepath.Join(outputDir, "images", "logo.png"))
	if err != nil {
		t.Fatal("images/logo.png not copied")
	}

	if string(content) != "PNG" {
		t.Errorf("logo.png content = %q", string(content))
	}

	// .gitkeep should be skipped
	if _, err := os.Stat(filepath.Join(outputDir, ".gitkeep")); err == nil {
		t.Error(".gitkeep should not be copied")
	}
}

func TestCopyStatic_NonexistentDir(t *testing.T) {
	t.Parallel()

	err := CopyStatic("/nonexistent/static", "/tmp/output")
	if err != nil {
		t.Errorf("CopyStatic should not error for nonexistent dir: %v", err)
	}
}
