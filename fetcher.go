package ssssg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
)

var (
	errHTTPStatus       = errors.New("unexpected HTTP status")
	errUnexpectedResult = errors.New("unexpected result type")
)

type Fetcher struct {
	baseDir string
	client  *http.Client
	mu      sync.Mutex
	cache   map[string]string
	group   singleflight.Group
}

func NewFetcher(baseDir string, client *http.Client) *Fetcher {
	if client == nil {
		client = http.DefaultClient
	}

	return &Fetcher{
		baseDir: baseDir,
		client:  client,
		cache:   make(map[string]string),
	}
}

func (f *Fetcher) Fetch(ctx context.Context, source string) (string, error) {
	f.mu.Lock()
	if v, ok := f.cache[source]; ok {
		f.mu.Unlock()

		return v, nil
	}
	f.mu.Unlock()

	v, err, _ := f.group.Do(source, func() (any, error) {
		var content string
		var fetchErr error

		if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
			content, fetchErr = f.fetchHTTP(ctx, source)
		} else {
			content, fetchErr = f.fetchFile(source)
		}

		if fetchErr != nil {
			return "", fetchErr
		}

		f.mu.Lock()
		f.cache[source] = content
		f.mu.Unlock()

		return content, nil
	})
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", source, err)
	}

	content, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("fetch %s: %w", source, errUnexpectedResult)
	}

	return content, nil
}

func (f *Fetcher) fetchHTTP(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request for %s: %w", url, err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch %s: %w: %d", url, errHTTPStatus, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response from %s: %w", url, err)
	}

	return string(body), nil
}

func (f *Fetcher) fetchFile(path string) (string, error) {
	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(f.baseDir, path)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", absPath, err)
	}

	return string(data), nil
}

func (f *Fetcher) ResolveFetchMap(ctx context.Context, fetchMap map[string]string) (map[string]string, error) {
	var mu sync.Mutex
	result := make(map[string]string, len(fetchMap))

	g, ctx := errgroup.WithContext(ctx)

	for key, source := range fetchMap {
		g.Go(func() error {
			content, err := f.Fetch(ctx, source)
			if err != nil {
				return fmt.Errorf("fetch %q: %w", key, err)
			}

			mu.Lock()
			result[key] = content
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("resolve fetch map: %w", err)
	}

	return result, nil
}
