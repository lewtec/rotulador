package annotation

import (
	"context"
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
		if err := f.Close(); err != nil {
			ReportError(context.Background(), err, "msg", "failed to close image file", "path", path)
		}
	}()
	m, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func IngestImage(img image.Image, outputDir string) error {
	tempFile := filepath.Join(outputDir, fmt.Sprintf("%s.png", uuid.New()))
	f, err := os.Create(tempFile)
	if err != nil {
		return err
	}

	hasher := sha256.New()
	w := io.MultiWriter(f, hasher)
	if err := png.Encode(w, img); err != nil {
		if closeErr := f.Close(); closeErr != nil {
			ReportError(context.Background(), closeErr, "msg", "failed to close temp image after encode error", "path", tempFile)
		}
		if removeErr := os.Remove(tempFile); removeErr != nil {
			ReportError(context.Background(), removeErr, "msg", "failed to remove temp file after encode error", "path", tempFile)
		}
		return err
	}

	if err := f.Close(); err != nil {
		if removeErr := os.Remove(tempFile); removeErr != nil {
			ReportError(context.Background(), removeErr, "msg", "failed to remove temp file after close error", "path", tempFile)
		}
		return err
	}

	finalPath := filepath.Join(outputDir, fmt.Sprintf("%x.png", hasher.Sum(nil)))
	if err := os.Rename(tempFile, finalPath); err != nil {
		if removeErr := os.Remove(tempFile); removeErr != nil {
			ReportError(context.Background(), removeErr, "msg", "failed to remove temp file", "path", tempFile)
		}
		return err
	}
	return nil
}
