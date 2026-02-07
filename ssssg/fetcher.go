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
)

var errHTTPStatus = errors.New("unexpected HTTP status")

type Fetcher struct {
	baseDir string
	client  *http.Client
	mu      sync.Mutex
	cache   map[string]string
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

	var content string
	var err error

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		content, err = f.fetchHTTP(ctx, source)
	} else {
		content, err = f.fetchFile(source)
	}

	if err != nil {
		return "", err
	}

	f.mu.Lock()
	f.cache[source] = content
	f.mu.Unlock()

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
	result := make(map[string]string, len(fetchMap))

	for key, source := range fetchMap {
		content, err := f.Fetch(ctx, source)
		if err != nil {
			return nil, fmt.Errorf("fetch %q: %w", key, err)
		}

		result[key] = content
	}

	return result, nil
}
