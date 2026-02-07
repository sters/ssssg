package ssssg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Check that expected files exist
	expectedFiles := []string{
		"site.yaml",
		"templates/_layout.html",
		"templates/_header.html",
		"templates/_footer.html",
		"templates/index.html",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(dir, f)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected file %s not found: %v", f, err)

			continue
		}

		if info.Size() == 0 {
			t.Errorf("file %s is empty", f)
		}
	}

	// Check static directory exists
	info, err := os.Stat(filepath.Join(dir, "static"))
	if err != nil {
		t.Errorf("static directory not found: %v", err)
	} else if !info.IsDir() {
		t.Error("static should be a directory")
	}
}

func TestInit_NoOverwrite(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Create a file that would be created by Init
	siteYAML := filepath.Join(dir, "site.yaml")
	if err := os.WriteFile(siteYAML, []byte("custom content"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Init(dir); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Check that existing file was not overwritten
	content, err := os.ReadFile(siteYAML)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "custom content" {
		t.Errorf("existing file was overwritten: got %q", string(content))
	}

	// But other files should be created
	_, err = os.Stat(filepath.Join(dir, "templates", "_layout.html"))
	if err != nil {
		t.Errorf("templates/_layout.html not created: %v", err)
	}
}
