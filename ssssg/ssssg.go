package ssssg

import (
	"context"
	"fmt"
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
}

func Build(ctx context.Context, opts BuildOptions) error {
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

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	client := &http.Client{Timeout: opts.Timeout}
	fetcher := NewFetcher(baseDir, client)

	// Resolve global fetch
	globalData := make(map[string]any)
	for k, v := range cfg.Global.Data {
		globalData[k] = v
	}

	if len(cfg.Global.Fetch) > 0 {
		fetched, err := fetcher.ResolveFetchMap(ctx, cfg.Global.Fetch)
		if err != nil {
			return fmt.Errorf("resolve global fetch: %w", err)
		}

		for k, v := range fetched {
			globalData[k] = v
		}
	}

	// Build each page
	for _, page := range cfg.Pages {
		pageData := make(map[string]any)
		for k, v := range page.Data {
			pageData[k] = v
		}

		if len(page.Fetch) > 0 {
			fetched, err := fetcher.ResolveFetchMap(ctx, page.Fetch)
			if err != nil {
				return fmt.Errorf("resolve fetch for %s: %w", page.Output, err)
			}

			for k, v := range fetched {
				pageData[k] = v
			}
		}

		data := TemplateData{
			Global: globalData,
			Page:   pageData,
		}

		if err := RenderPage(opts.TemplateDir, page, cfg.Global.Layout, data, opts.OutputDir); err != nil {
			return fmt.Errorf("render %s: %w", page.Output, err)
		}
	}

	// Copy static files
	if err := CopyStatic(opts.StaticDir, opts.OutputDir); err != nil {
		return fmt.Errorf("copy static: %w", err)
	}

	return nil
}
