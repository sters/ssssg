package ssssg

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetcher_FetchHTTP(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello from server"))
	}))
	defer srv.Close()

	f := NewFetcher("", srv.Client())
	content, err := f.Fetch(t.Context(), srv.URL)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if content != "hello from server" {
		t.Errorf("content = %q, want %q", content, "hello from server")
	}
}

func TestFetcher_FetchHTTP_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	f := NewFetcher("", srv.Client())
	_, err := f.Fetch(t.Context(), srv.URL)
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestFetcher_FetchFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("local content"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := NewFetcher(dir, nil)
	content, err := f.Fetch(t.Context(), "test.txt")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if content != "local content" {
		t.Errorf("content = %q, want %q", content, "local content")
	}
}

func TestFetcher_Cache(t *testing.T) {
	t.Parallel()

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		_, _ = w.Write([]byte("cached"))
	}))
	defer srv.Close()

	f := NewFetcher("", srv.Client())

	_, err := f.Fetch(t.Context(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.Fetch(t.Context(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (should be cached)", callCount)
	}
}

func TestFetcher_ResolveFetchMap(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("aaa"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("bbb"))
	}))
	defer srv.Close()

	f := NewFetcher(dir, srv.Client())
	result, err := f.ResolveFetchMap(t.Context(), map[string]string{
		"local":  "a.txt",
		"remote": srv.URL,
	})
	if err != nil {
		t.Fatalf("ResolveFetchMap failed: %v", err)
	}

	if result["local"] != "aaa" {
		t.Errorf("local = %q", result["local"])
	}

	if result["remote"] != "bbb" {
		t.Errorf("remote = %q", result["remote"])
	}
}

func TestFetcher_FetchFile_NotFound(t *testing.T) {
	t.Parallel()

	f := NewFetcher(t.TempDir(), nil)
	_, err := f.Fetch(t.Context(), "nonexistent.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
