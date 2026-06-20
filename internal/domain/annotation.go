package domain

import (
	"context"
	"time"
)

// Annotation represents a single classification/decision made by a user on an image.
// It serves as the core record mapping an image asset to a user's choice for a specific stage.
type Annotation struct {
	ID          int64
	ImageSHA256 string
	Username    string
	// StageIndex identifies the specific task/stage this annotation belongs to,
	// allowing for multi-step classification workflows.
	StageIndex  int
	OptionValue string
	AnnotatedAt time.Time
}

// AnnotationWithImage extends Annotation by joining the original image filename.
// This is typically used for presentation layers requiring both the annotation context
// and a human-readable reference to the image.
type AnnotationWithImage struct {
	Annotation
	ImageFilename string
}

// AnnotationStats provides aggregated, system-wide statistics about annotations.
// This is useful for dashboard overviews and tracking overall progress.
type AnnotationStats struct {
	AnnotatedImages  int64
	TotalAnnotations int64
	TotalUsers       int64
}

// AnnotationRepository defines the interface for annotation storage operations.
// Implementations abstract away the persistence details (e.g. SQL).
type AnnotationRepository interface {
	// Create registers a new annotation. If an annotation for the same image, user,
	// and stage already exists, it is expected to perform an "upsert", updating
	// the OptionValue and AnnotatedAt timestamp to reflect the user's latest choice.
	Create(ctx context.Context, imageSHA256 string, username string, stageIndex int, optionValue string) (*Annotation, error)

	// Get retrieves a specific annotation for a given image, user, and stage.
	// Implementations should return (nil, nil) if the annotation does not exist.
	Get(ctx context.Context, imageSHA256 string, username string, stageIndex int) (*Annotation, error)

	// GetForImage retrieves all annotations made by any user for a specific image.
	GetForImage(ctx context.Context, imageSHA256 string) ([]*Annotation, error)

	// GetByUser retrieves a paginated list of annotations created by a specific user.
	// It returns the joined AnnotationWithImage struct for easier rendering.
	GetByUser(ctx context.Context, username string, limit, offset int) ([]*AnnotationWithImage, error)

	// GetByImageAndUser retrieves all annotations for an image by a specific user
	GetByImageAndUser(ctx context.Context, imageSHA256 string, username string) ([]*Annotation, error)

	// CountByUser returns the total number of annotations by a user
	CountByUser(ctx context.Context, username string) (int64, error)

	// ListPendingImagesForUserAndStage finds images that need annotation by a user for a specific stage
	ListPendingImagesForUserAndStage(ctx context.Context, username string, stageIndex int, limit int) ([]*Image, error)

	// Exists checks if an annotation exists
	Exists(ctx context.Context, imageSHA256 string, username string, stageIndex int) (bool, error)

	// Delete removes an annotation by ID
	Delete(ctx context.Context, id int64) error

	// DeleteForImage removes all annotations for an image
	DeleteForImage(ctx context.Context, imageSHA256 string) error

	// GetStats returns overall annotation statistics
	GetStats(ctx context.Context) (*AnnotationStats, error)
}
