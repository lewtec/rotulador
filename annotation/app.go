package annotation

import (
	"database/sql"
	"log/slog"

	"github.com/lewtec/rotulador/internal/repository"
)

type AnnotatorApp struct {
	ImagesDir      string
	Database       *sql.DB
	Config         *Config
	Logger         *slog.Logger
	OffsetAdvance  int
	imageRepo      *repository.ImageRepository
	annotationRepo *repository.AnnotationRepository
}

type AnnotationStep struct {
	TaskID    string
	ImageID   string
	ImageName string
}

type TaskWithCount struct {
	*ConfigTask
	AvailableCount int
	TotalCount     int
	CompletedCount int
	PhaseProgress  *PhaseProgress
}

type PhaseProgress struct {
	Completed              int     // Images completed in this phase
	Pending                int     // Images eligible but not yet annotated
	FilteredWrongClass     int     // Images annotated in dependency phase but with wrong class
	NotYetAnnotated        int     // Images not yet annotated in dependency phase
	Total                  int     // Total images in the entire dataset
	CompletedPercent       float64 // Percentage of completed images
	PendingPercent         float64 // Percentage of pending images
	FilteredPercent        float64 // Percentage of filtered (wrong class) images
	NotYetAnnotatedPercent float64 // Percentage of not yet annotated images
}

type AnnotationResponse struct {
	ImageID string
	TaskID  string
	User    string
	Value   string
	Sure    bool
}

// ClassButton represents a class button with keyboard shortcut
type ClassButton struct {
	ID   string
	Name string
	Key  string
}
