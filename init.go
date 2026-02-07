package ssssg

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:scaffold/*
var scaffoldFS embed.FS

func Init(dir string) error {
	err := fs.WalkDir(scaffoldFS, "scaffold", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from "scaffold/" prefix
		relPath, err := filepath.Rel("scaffold", path)
		if err != nil {
			return fmt.Errorf("relative path: %w", err)
		}

		if relPath == "." {
			return nil
		}

		destPath := filepath.Join(dir, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}

		// Skip if file already exists
		if _, err := os.Stat(destPath); err == nil {
			return nil
		}

		data, err := scaffoldFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded file %s: %w", path, err)
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}

		if err := os.WriteFile(destPath, data, 0o644); err != nil { //nolint:gosec
			return fmt.Errorf("write %s: %w", destPath, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("walk scaffold: %w", err)
	}

	return nil
}
