package repository

import (
	"context"
	"database/sql"

	"github.com/lewtec/rotulador/internal/domain"
	"github.com/lewtec/rotulador/internal/sqlc"
)

// AnnotationRepository implements domain.AnnotationRepository using SQLC
type AnnotationRepository struct {
	queries *sqlc.Queries
}

// NewAnnotationRepository creates a new AnnotationRepository
func NewAnnotationRepository(db *sql.DB) *AnnotationRepository {
	return &AnnotationRepository{
		queries: sqlc.New(db),
	}
}

// NewAnnotationRepositoryWithTx creates a new AnnotationRepository with a transaction
func NewAnnotationRepositoryWithTx(tx *sql.Tx) *AnnotationRepository {
	return &AnnotationRepository{
		queries: sqlc.New(tx),
	}
}

// Create creates or updates an annotation (upsert)
func (r *AnnotationRepository) Create(ctx context.Context, imageSHA256 string, username string, stageIndex int, optionValue string) (*domain.Annotation, error) {
	params := sqlc.CreateAnnotationParams{
		ImageSha256: imageSHA256,
		Username:    username,
		StageIndex:  int64(stageIndex),
		OptionValue: optionValue,
	}

	ann, err := r.queries.CreateAnnotation(ctx, params)
	if err != nil {
		return nil, err
	}

	return toDomainAnnotation(ann), nil
}

// Get retrieves a specific annotation
func (r *AnnotationRepository) Get(ctx context.Context, imageSHA256 string, username string, stageIndex int) (*domain.Annotation, error) {
	params := sqlc.GetAnnotationParams{
		ImageSha256: imageSHA256,
		Username:    username,
		StageIndex:  int64(stageIndex),
	}

	ann, err := r.queries.GetAnnotation(ctx, params)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return toDomainAnnotation(ann), nil
}

// GetForImage retrieves all annotations for a specific image
func (r *AnnotationRepository) GetForImage(ctx context.Context, imageSHA256 string) ([]*domain.Annotation, error) {
	anns, err := r.queries.GetAnnotationsForImage(ctx, imageSHA256)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Annotation, len(anns))
	for i, ann := range anns {
		result[i] = toDomainAnnotation(ann)
	}

	return result, nil
}

// GetByUser retrieves annotations by a specific user (paginated)
func (r *AnnotationRepository) GetByUser(ctx context.Context, username string, limit, offset int) ([]*domain.AnnotationWithImage, error) {
	params := sqlc.GetAnnotationsByUserParams{
		Username: username,
		Limit:    int64(limit),
		Offset:   int64(offset),
	}

	rows, err := r.queries.GetAnnotationsByUser(ctx, params)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.AnnotationWithImage, len(rows))
	for i, row := range rows {
		ann := domain.AnnotationWithImage{
			Annotation: domain.Annotation{
				ID:          row.ID,
				ImageSHA256: row.ImageSha256,
				Username:    row.Username,
				StageIndex:  int(row.StageIndex),
				OptionValue: row.OptionValue,
			},
			ImageFilename: row.Filename,
		}
		if row.AnnotatedAt != nil {
			ann.AnnotatedAt = *row.AnnotatedAt
		}
		result[i] = &ann
	}

	return result, nil
}

// GetByImageAndUser retrieves all annotations for an image by a specific user
func (r *AnnotationRepository) GetByImageAndUser(ctx context.Context, imageSHA256 string, username string) ([]*domain.Annotation, error) {
	params := sqlc.GetAnnotationsByImageAndUserParams{
		ImageSha256: imageSHA256,
		Username:    username,
	}

	anns, err := r.queries.GetAnnotationsByImageAndUser(ctx, params)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Annotation, len(anns))
	for i, ann := range anns {
		result[i] = toDomainAnnotation(ann)
	}

	return result, nil
}

// CountByUser returns the total number of annotations by a user
func (r *AnnotationRepository) CountByUser(ctx context.Context, username string) (int64, error) {
	return r.queries.CountAnnotationsByUser(ctx, username)
}

// ListPendingImagesForUserAndStage finds images that need annotation by a user for a specific stage
func (r *AnnotationRepository) ListPendingImagesForUserAndStage(ctx context.Context, username string, stageIndex int, limit int) ([]*domain.Image, error) {
	params := sqlc.ListPendingImagesForUserAndStageParams{
		Username:   username,
		StageIndex: int64(stageIndex),
		Limit:      int64(limit),
	}

	images, err := r.queries.ListPendingImagesForUserAndStage(ctx, params)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Image, len(images))
	for i, img := range images {
		result[i] = toDomainImage(img)
	}

	return result, nil
}

// Exists checks if an annotation exists
func (r *AnnotationRepository) Exists(ctx context.Context, imageSHA256 string, username string, stageIndex int) (bool, error) {
	params := sqlc.CheckAnnotationExistsParams{
		ImageSha256: imageSHA256,
		Username:    username,
		StageIndex:  int64(stageIndex),
	}

	// The generated CheckAnnotationExists returns int64 (0 or 1 for SQLite)
	exists, err := r.queries.CheckAnnotationExists(ctx, params)
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// Delete removes an annotation by ID
func (r *AnnotationRepository) Delete(ctx context.Context, id int64) error {
	return r.queries.DeleteAnnotation(ctx, id)
}

// DeleteForImage removes all annotations for an image
func (r *AnnotationRepository) DeleteForImage(ctx context.Context, imageSHA256 string) error {
	return r.queries.DeleteAnnotationsForImage(ctx, imageSHA256)
}

// GetStats returns overall annotation statistics
func (r *AnnotationRepository) GetStats(ctx context.Context) (*domain.AnnotationStats, error) {
	stats, err := r.queries.GetAnnotationStats(ctx)
	if err != nil {
		return nil, err
	}

	return &domain.AnnotationStats{
		AnnotatedImages:  stats.AnnotatedImages,
		TotalAnnotations: stats.TotalAnnotations,
		TotalUsers:       stats.TotalUsers,
	}, nil
}

// toDomainAnnotation converts a sqlc.Annotation to domain.Annotation
func toDomainAnnotation(ann sqlc.Annotation) *domain.Annotation {
	d := &domain.Annotation{
		ID:          ann.ID,
		ImageSHA256: ann.ImageSha256,
		Username:    ann.Username,
		StageIndex:  int(ann.StageIndex),
		OptionValue: ann.OptionValue,
	}
	if ann.AnnotatedAt != nil {
		d.AnnotatedAt = *ann.AnnotatedAt
	}
	return d
}

// CountImagesWithoutAnnotationForStage counts images without any annotation for a stage
func (r *AnnotationRepository) CountImagesWithoutAnnotationForStage(ctx context.Context, stageIndex int64) (int64, error) {
	return r.queries.CountImagesWithoutAnnotationForStage(ctx, stageIndex)
}

// GetImageHashesWithAnnotation returns image SHA256 hashes that have a specific annotation value for a stage
func (r *AnnotationRepository) GetImageHashesWithAnnotation(ctx context.Context, stageIndex int64, optionValue string) ([]string, error) {
	params := sqlc.GetImageHashesWithAnnotationParams{
		StageIndex:  stageIndex,
		OptionValue: optionValue,
	}
	return r.queries.GetImageHashesWithAnnotation(ctx, params)
}

// CountPendingImagesForUserAndStage counts images needing annotation by a user for a specific stage
func (r *AnnotationRepository) CountPendingImagesForUserAndStage(ctx context.Context, username string, stageIndex int64) (int64, error) {
	params := sqlc.CountPendingImagesForUserAndStageParams{
		Username:   username,
		StageIndex: stageIndex,
	}
	return r.queries.CountPendingImagesForUserAndStage(ctx, params)
}

// CheckAnnotationExists checks if any annotation exists for an image at a stage (any user)
func (r *AnnotationRepository) CheckAnnotationExists(ctx context.Context, imageSHA256 string, username string, stageIndex int64) (bool, error) {
	// If username is empty, check if any annotation exists for this image+stage using optimized query
	if username == "" {
		params := sqlc.CheckAnnotationExistsForImageStageParams{
			ImageSha256: imageSHA256,
			StageIndex:  stageIndex,
		}
		exists, err := r.queries.CheckAnnotationExistsForImageStage(ctx, params)
		if err != nil {
			return false, err
		}
		return exists > 0, nil
	}
	// Otherwise use the specific user check
	return r.Exists(ctx, imageSHA256, username, int(stageIndex))
}

// Verify that AnnotationRepository implements domain.AnnotationRepository
var _ domain.AnnotationRepository = (*AnnotationRepository)(nil)
