package annotation

import (
	"crypto/sha256"
	"fmt"
	"github.com/google/uuid"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"path"
)

func DecodeImage(filepath string) (image.Image, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Printf("error closing image file: %v", err)
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
		if err := os.Remove(tempFile); err != nil {
			fmt.Printf("error removing temporary image file: %v", err)
		}
		return err
	}
	return err
}
