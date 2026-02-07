package ssssg

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type TemplateData struct {
	Global map[string]any
	Page   map[string]any
}

//nolint:gochecknoglobals
var funcMap = template.FuncMap{
	"raw": func(s string) string {
		return s
	},
}

var errNotDirectory = errors.New("path is not a directory")

func RenderPage(templateDir string, page PageConfig, globalLayout string, data TemplateData, outputDir string) error {
	tmpl := template.New("").Funcs(funcMap)

	// Parse all shared files (_*.html)
	sharedPattern := filepath.Join(templateDir, "_*.html")
	sharedFiles, err := filepath.Glob(sharedPattern)
	if err != nil {
		return fmt.Errorf("glob shared templates: %w", err)
	}

	if len(sharedFiles) > 0 {
		tmpl, err = tmpl.ParseFiles(sharedFiles...)
		if err != nil {
			return fmt.Errorf("parse shared templates: %w", err)
		}
	}

	// Parse the page template
	pageTemplatePath := filepath.Join(templateDir, page.Template)
	tmpl, err = tmpl.ParseFiles(pageTemplatePath)
	if err != nil {
		return fmt.Errorf("parse page template %s: %w", page.Template, err)
	}

	// Determine layout
	layout := page.Layout
	if layout == "" {
		layout = globalLayout
	}

	// Execute template
	var buf bytes.Buffer

	if layout != "" {
		err = tmpl.ExecuteTemplate(&buf, layout, data)
	} else {
		err = tmpl.ExecuteTemplate(&buf, filepath.Base(page.Template), data)
	}

	if err != nil {
		return fmt.Errorf("execute template for %s: %w", page.Output, err)
	}

	// Write output
	outputPath := filepath.Join(outputDir, page.Output)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	if err := os.WriteFile(outputPath, buf.Bytes(), 0o644); err != nil { //nolint:gosec
		return fmt.Errorf("write output %s: %w", outputPath, err)
	}

	return nil
}

func CopyStatic(staticDir, outputDir string) error {
	info, err := os.Stat(staticDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("stat static dir: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("static %s: %w", staticDir, errNotDirectory)
	}

	err = filepath.WalkDir(staticDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(staticDir, path)
		if err != nil {
			return fmt.Errorf("relative path: %w", err)
		}

		destPath := filepath.Join(outputDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}

		// Skip dotfiles like .gitkeep
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		return os.WriteFile(destPath, data, 0o644) //nolint:gosec
	})
	if err != nil {
		return fmt.Errorf("walk static dir: %w", err)
	}

	return nil
}
