package ssssg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCommand_Basic(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join(dir, "input.txt")
	dest := filepath.Join(dir, "output.txt")

	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	data := PipelineData{
		Src:  src,
		Dest: dest,
		Dir:  dir,
		Name: "input.txt",
		Ext:  ".txt",
		Base: "input",
	}

	if err := runCommand(t.Context(), "cp {{.Src}} {{.Dest}}", data); err != nil {
		t.Fatalf("runCommand failed: %v", err)
	}

	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal("output file not created")
	}

	if string(content) != "hello" {
		t.Errorf("content = %q, want %q", string(content), "hello")
	}
}

func TestRunCommand_Failure(t *testing.T) {
	t.Parallel()

	data := PipelineData{}

	err := runCommand(t.Context(), "false", data)
	if err == nil {
		t.Fatal("expected error for failing command")
	}
}

func TestRunCommand_AllTemplateVariables(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.txt")

	data := PipelineData{
		Src:  "/src/images/photo.jpg",
		Dest: "/dest/images/photo.jpg",
		Dir:  "/dest/images",
		Name: "photo.jpg",
		Ext:  ".jpg",
		Base: "photo",
	}

	cmdTmpl := `echo "{{.Src}}|{{.Dest}}|{{.Dir}}|{{.Name}}|{{.Ext}}|{{.Base}}" > ` + outFile

	if err := runCommand(t.Context(), cmdTmpl, data); err != nil {
		t.Fatalf("runCommand failed: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}

	got := strings.TrimSpace(string(content))
	want := "/src/images/photo.jpg|/dest/images/photo.jpg|/dest/images|photo.jpg|.jpg|photo"

	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMatchPipeline_BasenameMatch(t *testing.T) {
	t.Parallel()

	pipelines := []PipelineConfig{
		{Match: "*.jpg", Commands: []string{"echo jpg"}},
		{Match: "*.png", Commands: []string{"echo png"}},
	}

	p := matchPipeline("images/photo.jpg", pipelines)
	if p == nil {
		t.Fatal("expected match for *.jpg")
	}

	if p.Match != "*.jpg" {
		t.Errorf("matched %q, want %q", p.Match, "*.jpg")
	}
}

func TestMatchPipeline_RelativePathMatch(t *testing.T) {
	t.Parallel()

	pipelines := []PipelineConfig{
		{Match: "images/*.webp", Commands: []string{"echo webp"}},
	}

	p := matchPipeline("images/photo.webp", pipelines)
	if p == nil {
		t.Fatal("expected match for images/*.webp")
	}

	if p.Match != "images/*.webp" {
		t.Errorf("matched %q, want %q", p.Match, "images/*.webp")
	}

	// Should not match file in different directory
	p2 := matchPipeline("css/photo.webp", pipelines)
	if p2 != nil {
		t.Error("should not match file in different directory")
	}
}

func TestMatchPipeline_FirstMatchWins(t *testing.T) {
	t.Parallel()

	pipelines := []PipelineConfig{
		{Match: "*.jpg", Commands: []string{"first"}},
		{Match: "*.jpg", Commands: []string{"second"}},
	}

	p := matchPipeline("photo.jpg", pipelines)
	if p == nil {
		t.Fatal("expected match")
	}

	if p.Commands[0] != "first" {
		t.Errorf("commands[0] = %q, want %q", p.Commands[0], "first")
	}
}

func TestMatchPipeline_NoMatch(t *testing.T) {
	t.Parallel()

	pipelines := []PipelineConfig{
		{Match: "*.jpg", Commands: []string{"echo jpg"}},
	}

	p := matchPipeline("style.css", pipelines)
	if p != nil {
		t.Error("expected no match for style.css")
	}
}

func TestProcessStatic_WithPipeline(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	staticDir := filepath.Join(dir, "static")
	outputDir := filepath.Join(dir, "public")

	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(staticDir, "photo.jpg"), []byte("jpeg-data"), 0o644); err != nil {
		t.Fatal(err)
	}

	pipelines := []PipelineConfig{
		{
			Match: "*.jpg",
			Commands: []string{
				"cp {{.Src}} {{.Dest}}",
				"sh -c 'echo marker >> {{.Dest}}'",
			},
		},
	}

	if err := ProcessStatic(t.Context(), staticDir, outputDir, pipelines); err != nil {
		t.Fatalf("ProcessStatic failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "photo.jpg"))
	if err != nil {
		t.Fatal("output file not created")
	}

	got := string(content)
	if !strings.HasPrefix(got, "jpeg-data") {
		t.Errorf("missing original data in output")
	}

	if !strings.Contains(got, "marker") {
		t.Errorf("pipeline second command did not run; content = %q", got)
	}
}

func TestProcessStatic_UnmatchedFilesCopied(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	staticDir := filepath.Join(dir, "static")
	outputDir := filepath.Join(dir, "public")

	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(staticDir, "style.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	pipelines := []PipelineConfig{
		{Match: "*.jpg", Commands: []string{"cp {{.Src}} {{.Dest}}"}},
	}

	if err := ProcessStatic(t.Context(), staticDir, outputDir, pipelines); err != nil {
		t.Fatalf("ProcessStatic failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "style.css"))
	if err != nil {
		t.Fatal("unmatched file not copied")
	}

	if string(content) != "body{}" {
		t.Errorf("content = %q, want %q", string(content), "body{}")
	}
}

func TestProcessStatic_NoPipelines(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	staticDir := filepath.Join(dir, "static")
	outputDir := filepath.Join(dir, "public")

	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(staticDir, "style.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(staticDir, "app.js"), []byte("alert(1)"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ProcessStatic(t.Context(), staticDir, outputDir, nil); err != nil {
		t.Fatalf("ProcessStatic failed: %v", err)
	}

	css, err := os.ReadFile(filepath.Join(outputDir, "style.css"))
	if err != nil {
		t.Fatal("style.css not copied")
	}

	if string(css) != "body{}" {
		t.Errorf("style.css content = %q", string(css))
	}

	js, err := os.ReadFile(filepath.Join(outputDir, "app.js"))
	if err != nil {
		t.Fatal("app.js not copied")
	}

	if string(js) != "alert(1)" {
		t.Errorf("app.js content = %q", string(js))
	}
}
