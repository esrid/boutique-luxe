// Package media handles product image uploads: decode, cap dimensions,
// re-encode as JPEG. The decode/encode round-trip is what actually matters
// security-wise — it validates the upload is a real image (not a renamed
// script) and strips any EXIF/metadata payload along the way.
package media

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
)

var ErrInvalidImage = errors.New("media: not a valid image")

const maxDimension = 1600

// Save decodes r as an image, downscales it if either dimension exceeds
// maxDimension (preserving aspect ratio), and writes it as a JPEG under
// dir with a random filename. Returns that filename (not the full path).
func Save(dir string, r io.Reader) (string, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return "", ErrInvalidImage
	}
	img = downscale(img, maxDimension)

	name, err := randomName()
	if err != nil {
		return "", err
	}

	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return "", fmt.Errorf("create image file: %w", err)
	}
	defer f.Close()

	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 85}); err != nil {
		return "", fmt.Errorf("encode image: %w", err)
	}
	return name, nil
}

func downscale(img image.Image, maxDim int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxDim && h <= maxDim {
		return img
	}

	var newW, newH int
	if w > h {
		newW, newH = maxDim, h*maxDim/w
	} else {
		newH, newW = maxDim, w*maxDim/h
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	return dst
}

func randomName() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b) + ".jpg", nil
}
