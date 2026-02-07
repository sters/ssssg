package ssssg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"
)

type BuildOptions struct {
	ConfigPath  string
	TemplateDir string
	StaticDir   string
	OutputDir   string
	Timeout     time.Duration
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

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	client := &http.Client{Timeout: opts.Timeout}
	fetcher := NewFetcher(baseDir, client)

	// Collect all fetch sources (global + all pages) and resolve in parallel
	allFetch := make(map[string]string)
	for key, src := range cfg.Global.Fetch {
		allFetch[src] = src // deduplicate by source URL
		logf("Fetching global.%s: %s", key, src)
	}

	for _, page := range cfg.Pages {
		for key, src := range page.Fetch {
			allFetch[src] = src
			logf("Fetching %s.%s: %s", page.Output, key, src)
		}
	}

	if len(allFetch) > 0 {
		logf("Fetching %d source(s) in parallel...", len(allFetch))

		// Pre-fetch all sources in parallel (populates cache)
		if _, err := fetcher.ResolveFetchMap(ctx, allFetch); err != nil {
			return fmt.Errorf("resolve fetch: %w", err)
		}
	}

	// Build global data from data + cached fetch results
	globalData := make(map[string]any)
	for k, v := range cfg.Global.Data {
		globalData[k] = v
	}

	for key, src := range cfg.Global.Fetch {
		content, _ := fetcher.Fetch(ctx, src) // already cached
		globalData[key] = content
	}

	// Render each page in parallel
	logf("Building %d page(s)...", len(cfg.Pages))

	errs := make(chan error, len(cfg.Pages))

	for _, page := range cfg.Pages {
		go func(page PageConfig) {
			pageData := make(map[string]any)
			for k, v := range page.Data {
				pageData[k] = v
			}

			for key, src := range page.Fetch {
				content, _ := fetcher.Fetch(ctx, src) // already cached
				pageData[key] = content
			}

			data := TemplateData{
				Global: globalData,
				Page:   pageData,
			}

			if err := RenderPage(opts.TemplateDir, page, cfg.Global.Layout, data, opts.OutputDir); err != nil {
				errs <- fmt.Errorf("render %s: %w", page.Output, err)

				return
			}

			logf("  Generated: %s", page.Output)
			errs <- nil
		}(page)
	}

	for range len(cfg.Pages) {
		if err := <-errs; err != nil {
			return err
		}
	}

	// Copy static files
	logf("Copying static files...")

	if err := CopyStatic(opts.StaticDir, opts.OutputDir); err != nil {
		return fmt.Errorf("copy static: %w", err)
	}

	logf("Done!")

	return nil
}
