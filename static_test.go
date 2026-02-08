package ssssg

import (
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/webp"
)

// createTestImage writes a minimal image of the given size in the specified format.
func createTestImage(t *testing.T, path string, w, h int, format string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.White)

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	switch format {
	case "png":
		err = png.Encode(f, img)
	case "jpeg":
		err = jpeg.Encode(f, img, nil)
	case "gif":
		err = gif.Encode(f, img, nil)
	default:
		t.Fatalf("unsupported test image format: %s", format)
	}

	if err != nil {
		t.Fatal(err)
	}
}

// Ignore the unused import lint — webp is used only for blank-import registration
// via static.go, but we import it here to ensure the test binary links the decoder.
var _ = webp.Decode

func TestScanStaticFiles_Images(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	createTestImage(t, filepath.Join(dir, "photo.png"), 100, 200, "png")
	createTestImage(t, filepath.Join(dir, "pic.jpg"), 640, 480, "jpeg")
	createTestImage(t, filepath.Join(dir, "anim.gif"), 32, 32, "gif")

	result, err := ScanStaticFiles(dir, 2)
	if err != nil {
		t.Fatalf("ScanStaticFiles: %v", err)
	}

	tests := []struct {
		key          string
		wantW, wantH int
		wantSizeGt0  bool
	}{
		{"photo.png", 100, 200, true},
		{"pic.jpg", 640, 480, true},
		{"anim.gif", 32, 32, true},
	}

	for _, tt := range tests {
		si, ok := result[tt.key]
		if !ok {
			t.Errorf("missing key %q", tt.key)

			continue
		}

		if si.Width != tt.wantW || si.Height != tt.wantH {
			t.Errorf("%s: got %dx%d, want %dx%d", tt.key, si.Width, si.Height, tt.wantW, tt.wantH)
		}

		if si.Path != tt.key {
			t.Errorf("%s: Path = %q, want %q", tt.key, si.Path, tt.key)
		}

		if tt.wantSizeGt0 && si.Size <= 0 {
			t.Errorf("%s: Size = %d, want > 0", tt.key, si.Size)
		}
	}
}

func TestScanStaticFiles_NonImage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	data := []byte("body { margin: 0; }")
	if err := os.WriteFile(filepath.Join(dir, "style.css"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ScanStaticFiles(dir, 2)
	if err != nil {
		t.Fatalf("ScanStaticFiles: %v", err)
	}

	si, ok := result["style.css"]
	if !ok {
		t.Fatal("missing key style.css")
	}

	if si.Path != "style.css" {
		t.Errorf("Path = %q", si.Path)
	}

	if si.Size != int64(len(data)) {
		t.Errorf("Size = %d, want %d", si.Size, len(data))
	}

	if si.Width != 0 || si.Height != 0 {
		t.Errorf("non-image should have 0x0, got %dx%d", si.Width, si.Height)
	}
}

func TestScanStaticFiles_BrokenImage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Write garbage data with an image extension
	if err := os.WriteFile(filepath.Join(dir, "bad.png"), []byte("not a png"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ScanStaticFiles(dir, 2)
	if err != nil {
		t.Fatalf("ScanStaticFiles: %v", err)
	}

	si := result["bad.png"]
	if si.Width != 0 || si.Height != 0 {
		t.Errorf("broken image should have 0x0, got %dx%d", si.Width, si.Height)
	}

	if si.Path != "bad.png" {
		t.Errorf("Path = %q", si.Path)
	}
}

func TestScanStaticFiles_EmptyDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	result, err := ScanStaticFiles(dir, 2)
	if err != nil {
		t.Fatalf("ScanStaticFiles: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}

func TestScanStaticFiles_NonExistentDir(t *testing.T) {
	t.Parallel()

	result, err := ScanStaticFiles(filepath.Join(t.TempDir(), "does-not-exist"), 2)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestScanStaticFiles_DotfileSkipped(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, ".gitkeep"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "visible.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ScanStaticFiles(dir, 2)
	if err != nil {
		t.Fatalf("ScanStaticFiles: %v", err)
	}

	if _, ok := result[".gitkeep"]; ok {
		t.Error("dotfile should be skipped")
	}

	if _, ok := result["visible.txt"]; !ok {
		t.Error("visible.txt should be present")
	}
}

func TestScanStaticFiles_Subdirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	imgDir := filepath.Join(dir, "img")
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	createTestImage(t, filepath.Join(imgDir, "photo.png"), 50, 75, "png")

	result, err := ScanStaticFiles(dir, 2)
	if err != nil {
		t.Fatalf("ScanStaticFiles: %v", err)
	}

	si, ok := result["img/photo.png"]
	if !ok {
		t.Fatal("missing key img/photo.png")
	}

	if si.Path != "img/photo.png" {
		t.Errorf("Path = %q, want %q", si.Path, "img/photo.png")
	}

	if si.Width != 50 || si.Height != 75 {
		t.Errorf("got %dx%d, want 50x75", si.Width, si.Height)
	}
}
