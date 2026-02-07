package ssssg

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "golang.org/x/image/webp"
)

// ScanStaticFiles walks dir and returns metadata for every file found.
// Image files (.jpg, .jpeg, .png, .gif, .webp) have Width/Height populated
// via image.DecodeConfig (header-only, fast). Errors on individual files are
// silently ignored so that a broken image never stops the build.
func ScanStaticFiles(dir string) (map[string]StaticFileInfo, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
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
	var wg sync.WaitGroup

	for _, f := range files {
		wg.Add(1)

		go func() {
			defer wg.Done()

			si := scanFile(f.path, f.relPath)

			mu.Lock()
			result[filepath.ToSlash(f.relPath)] = si
			mu.Unlock()
		}()
	}

	wg.Wait()

	return result, nil
}

// imageExts lists extensions handled by the registered image decoders.
var imageExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
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
	if !imageExts[ext] {
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
