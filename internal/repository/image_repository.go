package repository

import (
	"context"
	"database/sql"

	"github.com/lewtec/rotulador/internal/db/sqlc"
	"github.com/lewtec/rotulador/internal/domain"
)

// ImageRepository implements domain.ImageRepository using SQLC
type ImageRepository struct {
	queries *sqlc.Queries
}

// NewImageRepository creates a new ImageRepository
func NewImageRepository(db *sql.DB) *ImageRepository {
	return &ImageRepository{
		queries: sqlc.New(db),
	}
}

// NewImageRepositoryWithTx creates a new ImageRepository with a transaction
func NewImageRepositoryWithTx(tx *sql.Tx) *ImageRepository {
	return &ImageRepository{
		queries: sqlc.New(tx),
	}
}

// Create creates a new image record
func (r *ImageRepository) Create(ctx context.Context, sha256, filename string) (*domain.Image, error) {
	img, err := r.queries.CreateImage(ctx, sqlc.CreateImageParams{
		Sha256:   sha256,
		Filename: filename,
	})
	return convertRow(img, err, toDomainImage)
}

// GetBySHA256 retrieves an image by its SHA256 hash
func (r *ImageRepository) GetBySHA256(ctx context.Context, sha256 string) (*domain.Image, error) {
	img, err := r.queries.GetImage(ctx, sha256)
	return convertMissingRow(img, err, toDomainImage)
}

// GetByFilename retrieves an image by its filename
func (r *ImageRepository) GetByFilename(ctx context.Context, filename string) (*domain.Image, error) {
	img, err := r.queries.GetImageByFilename(ctx, filename)
	return convertMissingRow(img, err, toDomainImage)
}

// List retrieves all images
func (r *ImageRepository) List(ctx context.Context) ([]*domain.Image, error) {
	images, err := r.queries.ListImages(ctx)
	return convertRows(images, err, toDomainImage)
}

// Count returns the total number of images
func (r *ImageRepository) Count(ctx context.Context) (int64, error) {
	return r.queries.CountImages(ctx)
}

// Delete removes an image by SHA256 hash
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
