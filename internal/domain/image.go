package domain

import (
	"context"
	"time"
)

// Image represents an image to be annotated
type Image struct {
	SHA256     string
	Filename   string
	IngestedAt time.Time
}

// ImageRepository defines the interface for image storage operations
type ImageRepository interface {
	// Create creates a new image record
	Create(ctx context.Context, sha256, filename string) (*Image, error)

	// GetBySHA256 retrieves an image by its SHA256 hash.
	// Missing rows return an error that errors.Is matches sql.ErrNoRows.
	GetBySHA256(ctx context.Context, sha256 string) (*Image, error)

	// GetByFilename retrieves an image by its filename.
	// Missing rows return an error that errors.Is matches sql.ErrNoRows.
	GetByFilename(ctx context.Context, filename string) (*Image, error)

	// List retrieves all images
	List(ctx context.Context) ([]*Image, error)

	// Count returns the total number of images
	Count(ctx context.Context) (int64, error)

	// Delete removes an image by SHA256
	Delete(ctx context.Context, sha256 string) error
}
