package ssssg

import (
	"fmt"
	"image"
	_ "image/gif"  // Register GIF decoder.
	_ "image/jpeg" // Register JPEG decoder.
	_ "image/png"  // Register PNG decoder.
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "golang.org/x/image/webp" // Register WebP decoder.
	"golang.org/x/sync/errgroup"
)

// ScanStaticFiles walks dir and returns metadata for every file found.
// Image files (.jpg, .jpeg, .png, .gif, .webp) have Width/Height populated
// via image.DecodeConfig (header-only, fast). Errors on individual files are
// silently ignored so that a broken image never stops the build.
func ScanStaticFiles(dir string, parallelism int) (map[string]StaticFileInfo, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]StaticFileInfo{}, nil
		}

		return nil, fmt.Errorf("stat dir: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%s: %w", dir, errNotDirectory)
	}

	type fileEntry struct {
		path    string
		relPath string
	}

	var files []fileEntry

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Skip dotfiles
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("relative path: %w", err)
		}

		files = append(files, fileEntry{path: path, relPath: relPath})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk dir: %w", err)
	}

	result := make(map[string]StaticFileInfo, len(files))
	var mu sync.Mutex

	g := new(errgroup.Group)
	g.SetLimit(parallelism)

	for _, f := range files {
		g.Go(func() error {
			si := scanFile(f.path, f.relPath)

			mu.Lock()
			result[filepath.ToSlash(f.relPath)] = si
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("scan files: %w", err)
	}

	return result, nil
}

// isImageExt reports whether ext is handled by the registered image decoders.
func isImageExt(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	}

	return false
}

// scanFile returns metadata for a single file. Decode errors are swallowed.
func scanFile(absPath, relPath string) StaticFileInfo {
	si := StaticFileInfo{
		Path: filepath.ToSlash(relPath),
	}

	fi, err := os.Stat(absPath)
	if err != nil {
		return si
	}

	si.Size = fi.Size()

	ext := strings.ToLower(filepath.Ext(absPath))
	if !isImageExt(ext) {
		return si
	}

	f, err := os.Open(absPath)
	if err != nil {
		return si
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return si
	}

	si.Width = cfg.Width
	si.Height = cfg.Height

	return si
}
