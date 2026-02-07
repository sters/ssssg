package ssssg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sync/errgroup"
)

type BuildOptions struct {
	ConfigPath  string
	TemplateDir string
	StaticDir   string
	OutputDir   string
	Timeout     time.Duration
	Clean       bool
	Log         io.Writer
}

func Build(ctx context.Context, opts BuildOptions) error {
	logf := func(_ string, _ ...any) {}
	if opts.Log != nil {
		logf = func(format string, args ...any) {
			fmt.Fprintf(opts.Log, format+"\n", args...)
		}
	}

	logf("Loading config: %s", opts.ConfigPath)

	cfg, err := LoadConfig(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	baseDir := filepath.Dir(opts.ConfigPath)

	if opts.TemplateDir == "" {
		opts.TemplateDir = filepath.Join(baseDir, "templates")
	}

	if opts.StaticDir == "" {
		opts.StaticDir = filepath.Join(baseDir, "static")
	}

	if opts.OutputDir == "" {
		opts.OutputDir = filepath.Join(baseDir, "public")
	}

	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}

	logf("Templates: %s", opts.TemplateDir)
	logf("Output:    %s", opts.OutputDir)

	// Clean output directory if requested
	if opts.Clean {
		logf("Cleaning output directory...")

		if err := os.RemoveAll(opts.OutputDir); err != nil {
			return fmt.Errorf("clean output dir: %w", err)
		}
	}

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	// HTTP client relies on context for timeout — no separate client timeout
	fetcher := NewFetcher(baseDir, &http.Client{})

	// Collect all unique fetch sources
	sources := make(map[string]struct{})
	for key, src := range cfg.Global.Fetch {
		sources[src] = struct{}{}
		logf("Fetching global.%s: %s", key, src)
	}

	for _, page := range cfg.Pages {
		for key, src := range page.Fetch {
			sources[src] = struct{}{}
			logf("Fetching %s.%s: %s", page.Output, key, src)
		}
	}

	// Prefetch all unique sources in parallel
	if len(sources) > 0 {
		logf("Fetching %d source(s) in parallel...", len(sources))

		g, gctx := errgroup.WithContext(ctx)
		for src := range sources {
			g.Go(func() error {
				_, err := fetcher.Fetch(gctx, src)

				return err
			})
		}

		if err := g.Wait(); err != nil {
			return fmt.Errorf("fetch: %w", err)
		}
	}

	// Build global data from data + cached fetch results
	globalData := make(map[string]any)
	for k, v := range cfg.Global.Data {
		globalData[k] = v
	}

	for key, src := range cfg.Global.Fetch {
		content, err := fetcher.Fetch(ctx, src)
		if err != nil {
			return fmt.Errorf("resolve global fetch %q: %w", key, err)
		}

		globalData[key] = content
	}

	// Render each page in parallel
	logf("Building %d page(s)...", len(cfg.Pages))

	g, gctx := errgroup.WithContext(ctx)

	for _, page := range cfg.Pages {
		g.Go(func() error {
			pageData := make(map[string]any)
			for k, v := range page.Data {
				pageData[k] = v
			}

			for key, src := range page.Fetch {
				content, err := fetcher.Fetch(gctx, src)
				if err != nil {
					return fmt.Errorf("resolve %s fetch %q: %w", page.Output, key, err)
				}

				pageData[key] = content
			}

			data := TemplateData{
				Global: globalData,
				Page:   pageData,
			}

			if err := RenderPage(opts.TemplateDir, page, cfg.Global.Layout, data, opts.OutputDir); err != nil {
				return fmt.Errorf("render %s: %w", page.Output, err)
			}

			logf("  Generated: %s", page.Output)

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("build pages: %w", err)
	}

	// Process static files
	logf("Processing static files...")

	if err := ProcessStatic(ctx, opts.StaticDir, opts.OutputDir, cfg.Static.Pipelines); err != nil {
		return fmt.Errorf("process static: %w", err)
	}

	logf("Done!")

	return nil
}
