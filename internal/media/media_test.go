package media_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/esrid/maison/internal/media"
)

func encodedPNG(t *testing.T, w, h int) *bytes.Buffer {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 100, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode test png: %v", err)
	}
	return &buf
}

func TestSave_SmallImagePassesThrough(t *testing.T) {
	dir := t.TempDir()
	name, err := media.Save(dir, encodedPNG(t, 100, 80))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !strings.HasSuffix(name, ".jpg") {
		t.Errorf("filename = %q, want .jpg suffix", name)
	}

	f, err := os.Open(dir + "/" + name)
	if err != nil {
		t.Fatalf("open saved file: %v", err)
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatalf("decode saved image: %v", err)
	}
	if cfg.Width != 100 || cfg.Height != 80 {
		t.Errorf("dimensions = %dx%d, want 100x80 (below max, should pass through unscaled)", cfg.Width, cfg.Height)
	}
}

func TestSave_LargeImageIsDownscaled(t *testing.T) {
	dir := t.TempDir()
	name, err := media.Save(dir, encodedPNG(t, 3000, 1500))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	f, err := os.Open(dir + "/" + name)
	if err != nil {
		t.Fatalf("open saved file: %v", err)
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatalf("decode saved image: %v", err)
	}
	if cfg.Width != 1600 {
		t.Errorf("width = %d, want 1600 (capped)", cfg.Width)
	}
	if cfg.Height != 800 {
		t.Errorf("height = %d, want 800 (aspect ratio preserved: 1500 * 1600/3000)", cfg.Height)
	}
}

func TestSave_RejectsNonImageInput(t *testing.T) {
	dir := t.TempDir()
	_, err := media.Save(dir, strings.NewReader("not an image, just text pretending to be one"))
	if err != media.ErrInvalidImage {
		t.Fatalf("err = %v, want ErrInvalidImage", err)
	}
}
