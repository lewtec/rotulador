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
	"path"

	"github.com/google/uuid"
)

func DecodeImage(filepath string) (image.Image, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			ReportError(context.TODO(), err, "msg", "failed to close image file", "path", filepath)
		}
	}()
	m, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return m, err
}

func IngestImage(img image.Image, outputDir string) error {
	tempFile := path.Join(outputDir, fmt.Sprintf("%s.png", uuid.New()))
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
	err = os.Rename(tempFile, path.Join(outputDir, fmt.Sprintf("%x.png", hasher.Sum(nil))))
	if err != nil {
		if removeErr := os.Remove(tempFile); removeErr != nil {
			ReportError(context.TODO(), removeErr, "msg", "failed to remove temp file", "path", tempFile)
		}
		return err
	}
	return err
}
