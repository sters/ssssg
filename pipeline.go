package ssssg

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/sync/errgroup"
)

// PipelineData holds template variables available in pipeline command strings.
type PipelineData struct {
	Src  string // Source file absolute path
	Dest string // Destination file absolute path
	Dir  string // Destination directory
	Name string // File name
	Ext  string // File extension
	Base string // File name without extension
}

// ProcessStatic walks the static directory and processes each file.
// Files matching a pipeline have their commands executed in order.
// Unmatched files are copied using copyFile.
func ProcessStatic(ctx context.Context, staticDir, outputDir string, pipelines []PipelineConfig) error {
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

	type fileEntry struct {
		path    string
		relPath string
	}

	var files []fileEntry

	err = filepath.WalkDir(staticDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Skip dotfiles like .gitkeep
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		relPath, err := filepath.Rel(staticDir, path)
		if err != nil {
			return fmt.Errorf("relative path: %w", err)
		}

		files = append(files, fileEntry{path: path, relPath: relPath})

		return nil
	})
	if err != nil {
		return fmt.Errorf("walk static dir: %w", err)
	}

	g, gctx := errgroup.WithContext(ctx)

	for _, f := range files {
		g.Go(func() error {
			destPath := filepath.Join(outputDir, f.relPath)

			if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
				return fmt.Errorf("create dir for %s: %w", f.relPath, err)
			}

			pipeline := matchPipeline(f.relPath, pipelines)
			if pipeline == nil {
				return copyFile(f.path, destPath)
			}

			data := PipelineData{
				Src:  f.path,
				Dest: destPath,
				Dir:  filepath.Dir(destPath),
				Name: filepath.Base(f.path),
				Ext:  filepath.Ext(f.path),
				Base: strings.TrimSuffix(filepath.Base(f.path), filepath.Ext(f.path)),
			}

			return runPipeline(gctx, pipeline, data)
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("process static files: %w", err)
	}

	return nil
}

// matchPipeline returns the first pipeline whose pattern matches the file.
// If the pattern contains '/', it matches against the relative path.
// Otherwise, it matches against the basename.
func matchPipeline(relPath string, pipelines []PipelineConfig) *PipelineConfig {
	for i := range pipelines {
		p := &pipelines[i]
		name := relPath

		if !strings.Contains(p.Match, "/") {
			name = filepath.Base(relPath)
		}

		matched, err := filepath.Match(p.Match, name)
		if err != nil {
			continue
		}

		if matched {
			return p
		}
	}

	return nil
}

// runPipeline executes each command in a pipeline sequentially.
func runPipeline(ctx context.Context, pipeline *PipelineConfig, data PipelineData) error {
	for _, cmdTmpl := range pipeline.Commands {
		if err := runCommand(ctx, cmdTmpl, data); err != nil {
			return err
		}
	}

	return nil
}

// runCommand renders a command template with PipelineData and executes it via sh -c.
func runCommand(ctx context.Context, cmdTemplate string, data PipelineData) error {
	tmpl, err := template.New("cmd").Parse(cmdTemplate)
	if err != nil {
		return fmt.Errorf("parse command template %q: %w", cmdTemplate, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute command template %q: %w", cmdTemplate, err)
	}

	rendered := buf.String()

	cmd := exec.CommandContext(ctx, "sh", "-c", rendered)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command %q: %w", rendered, err)
	}

	return nil
}
