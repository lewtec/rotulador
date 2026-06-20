package repository

import (
	"context"
	"database/sql"

	"github.com/lewtec/rotulador/internal/domain"
	"github.com/lewtec/rotulador/internal/sqlc"
)

// ImageRepository implements domain.ImageRepository using SQLC.
// It wraps generated SQLC queries to map them to the domain model.
type ImageRepository struct {
	queries *sqlc.Queries
}

// NewImageRepository creates a new ImageRepository using a standard database connection.
func NewImageRepository(db *sql.DB) *ImageRepository {
	return &ImageRepository{
		queries: sqlc.New(db),
	}
}

// NewImageRepositoryWithTx creates a new ImageRepository bound to an active transaction.
// This is required when image operations must be atomically committed alongside other changes.
func NewImageRepositoryWithTx(tx *sql.Tx) *ImageRepository {
	return &ImageRepository{
		queries: sqlc.New(tx),
	}
}

// Create registers a new image record in the database.
// It relies on the database schema's UNIQUE constraint on the sha256 column
// to prevent duplicate insertions.
func (r *ImageRepository) Create(ctx context.Context, sha256, filename string) (*domain.Image, error) {
	params := sqlc.CreateImageParams{
		Sha256:   sha256,
		Filename: filename,
	}

	img, err := r.queries.CreateImage(ctx, params)
	if err != nil {
		return nil, err
	}

	return toDomainImage(img), nil
}

// GetBySHA256 retrieves an image by its SHA256 hash identifier.
// It intercepts sql.ErrNoRows to return (nil, nil) instead of an error,
// explicitly modeling the "not found" state in the domain logic.
func (r *ImageRepository) GetBySHA256(ctx context.Context, sha256 string) (*domain.Image, error) {
	img, err := r.queries.GetImage(ctx, sha256)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return toDomainImage(img), nil
}

// GetByFilename retrieves an image by its original filename.
// Like GetBySHA256, it returns (nil, nil) if no matching record is found.
func (r *ImageRepository) GetByFilename(ctx context.Context, filename string) (*domain.Image, error) {
	img, err := r.queries.GetImageByFilename(ctx, filename)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return toDomainImage(img), nil
}

// List retrieves all ingested images from the database.
// Note: It loads all records into memory. For large datasets, this could lead to high
// memory consumption and should be paginated at the database query level.
func (r *ImageRepository) List(ctx context.Context) ([]*domain.Image, error) {
	images, err := r.queries.ListImages(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Image, len(images))
	for i, img := range images {
		result[i] = toDomainImage(img)
	}

	return result, nil
}

// Count returns the total number of images present in the repository.
func (r *ImageRepository) Count(ctx context.Context) (int64, error) {
	return r.queries.CountImages(ctx)
}

// Delete removes an image record by its SHA256 hash.
// This executes a hard delete. Depending on foreign key constraints (e.g. ON DELETE CASCADE),
// this may also remove related annotations.
func (r *ImageRepository) Delete(ctx context.Context, sha256 string) error {
	return r.queries.DeleteImage(ctx, sha256)
}

// toDomainImage converts a sqlc.Image to domain.Image
func toDomainImage(img sqlc.Image) *domain.Image {
	d := &domain.Image{
		SHA256:   img.Sha256,
		Filename: img.Filename,
	}
	if img.IngestedAt != nil {
		d.IngestedAt = *img.IngestedAt
	}
	return d
}

// Verify that ImageRepository implements domain.ImageRepository
var _ domain.ImageRepository = (*ImageRepository)(nil)
