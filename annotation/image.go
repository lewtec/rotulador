package annotation

import (
	"crypto/sha256"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

func DecodeImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()
	m, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return m, err
}

func IngestImage(img image.Image, outputDir string) error {
	tempFile := filepath.Join(outputDir, fmt.Sprintf("%s.png", uuid.New()))
	f, err := os.Create(tempFile)
	if err != nil {
		return err
	}
	hasher := sha256.New()
	w := io.MultiWriter(f, hasher)
	err = png.Encode(w, img)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}
	err = os.Rename(tempFile, filepath.Join(outputDir, fmt.Sprintf("%x.png", hasher.Sum(nil))))
	if err != nil {
		_ = os.Remove(tempFile)
		return err
	}
	return err
}
