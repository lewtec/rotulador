package domain

import (
	"context"
	"time"
)

// Image represents an image to be annotated.
// The SHA256 hash serves as the primary identifier across the system,
// ensuring deduplication of ingested assets.
type Image struct {
	SHA256     string
	Filename   string
	// IngestedAt records when the image was first added to the system,
	// useful for auditing and sorting pending tasks.
	IngestedAt time.Time
}

// ImageRepository defines the interface for image storage operations.
// Implementations should handle mapping between the domain models and
// the underlying persistence layer.
type ImageRepository interface {
	// Create registers a new image record in the database.
	// It is expected to enforce uniqueness on the SHA256 hash, potentially
	// returning an error if a duplicate is inserted.
	Create(ctx context.Context, sha256, filename string) (*Image, error)

	// GetBySHA256 retrieves a single image by its SHA256 hash identifier.
	// Implementations should return (nil, nil) if the image is not found,
	// rather than an error, to signify an expected empty state.
	GetBySHA256(ctx context.Context, sha256 string) (*Image, error)

	// GetByFilename retrieves a single image by its original filename.
	// Similar to GetBySHA256, it returns (nil, nil) if no match is found.
	GetByFilename(ctx context.Context, filename string) (*Image, error)

	// List retrieves all ingested images.
	// Note: This loads all records into memory and should be paginated
	// if the dataset scales beyond typical in-memory capacities.
	List(ctx context.Context) ([]*Image, error)

	// Count returns the total number of images currently tracked in the repository.
	Count(ctx context.Context) (int64, error)

	// Delete removes an image and its associated records by its SHA256 hash.
	// This operation is typically irreversible and requires cascading deletions
	// for dependent records like annotations.
	Delete(ctx context.Context, sha256 string) error
}
