package annotation

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"

	"github.com/lewtec/rotulador/internal/domain"
)

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

// getCachedImageList returns the list of all images, using cache if available
func (a *AnnotatorApp) getCachedImageList(ctx context.Context) ([]*domain.Image, error) {
	// Try to get from cache first
	if cache := GetRequestCache(ctx); cache != nil {
		if images, ok := cache.GetImages(); ok {
			return images, nil
		}
	}

	// Cache miss or no cache available, fetch from database
	images, err := a.imageRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	// Store in cache if available
	if cache := GetRequestCache(ctx); cache != nil {
		cache.SetImages(images)
	}

	return images, nil
}

// CountEligibleImages counts all images that are eligible for this task (regardless of annotation status)
func (a *AnnotatorApp) CountEligibleImages(ctx context.Context, taskID string) (int, error) {
	// Find stage index for this task
	stageIndex := a.findTaskIndex(taskID)
	if stageIndex == -1 {
		return 0, fmt.Errorf("task not found: %s", taskID)
	}

	task := a.Config.Tasks[stageIndex]

	// If no dependencies, all images are eligible
	if len(task.If) == 0 {
		count, err := a.imageRepo.Count(ctx)
		return int(count), err
	}

	// Pre-fetch all dependency data before looping (optimization: move queries outside loop)
	imageHashesByDep, err := a.getDependencyImageHashes(ctx, task)
	if err != nil {
		return 0, err
	}

	// Get all images and filter by dependencies (using cache)
	allImages, err := a.getCachedImageList(ctx)
	if err != nil {
		return 0, fmt.Errorf("while listing images: %w", err)
	}

	validCount := 0
	for _, img := range allImages {
		valid := true
		// Check each dependency using pre-fetched map
		for depTaskID := range task.If {
			if !imageHashesByDep[depTaskID][img.SHA256] {
				valid = false
				break
			}
		}

		if valid {
			validCount++
		}
	}

	return validCount, nil
}

func (a *AnnotatorApp) CountAvailableImages(ctx context.Context, taskID string) (int, error) {
	// Find stage index for this task
	stageIndex := a.findTaskIndex(taskID)
	if stageIndex == -1 {
		return 0, fmt.Errorf("task not found: %s", taskID)
	}

	task := a.Config.Tasks[stageIndex]

	// Count images without annotation for this stage
	count, err := a.annotationRepo.CountImagesWithoutAnnotationForStage(ctx, int64(stageIndex))
	if err != nil {
		return 0, fmt.Errorf("while counting available images: %w", err)
	}

	// Handle task dependencies (If field)
	// If there are dependencies, we need to filter images that meet the criteria
	if len(task.If) > 0 {
		// Pre-fetch all dependency data before looping (optimization: move queries outside loop)
		imageHashesByDep, err := a.getDependencyImageHashes(ctx, task)
		if err != nil {
			return 0, err
		}

		// Get all candidate images (using cache)
		allImages, err := a.getCachedImageList(ctx)
		if err != nil {
			return 0, fmt.Errorf("while listing images: %w", err)
		}

		validCount := 0
		for _, img := range allImages {
			valid := true
			// Check each dependency using pre-fetched map
			for depTaskID := range task.If {
				if !imageHashesByDep[depTaskID][img.SHA256] {
					valid = false
					break
				}
			}

			if valid {
				// Check if this image has annotation for current stage
				hasAnnotation, err := a.annotationRepo.CheckAnnotationExists(ctx, img.SHA256, "", int64(stageIndex))
				if err != nil {
					return 0, err
				}
				if !hasAnnotation {
					validCount++
				}
			}
		}
		return validCount, nil
	}

	return int(count), nil
}

// GetPhaseProgressStats calculates comprehensive progress statistics for a task
func (a *AnnotatorApp) GetPhaseProgressStats(ctx context.Context, taskID string) (*PhaseProgress, error) {
	// Get total images in the entire dataset
	totalCount, err := a.imageRepo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("while counting total images: %w", err)
	}

	// Get eligible images (that pass filters from previous phases)
	eligibleCount, err := a.CountEligibleImages(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("while counting eligible images: %w", err)
	}

	// Get available images (eligible but not yet annotated)
	availableCount, err := a.CountAvailableImages(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("while counting available images: %w", err)
	}

	// Calculate completed and pending
	completed := eligibleCount - availableCount
	if completed < 0 {
		completed = 0
	}
	pending := availableCount

	total := int(totalCount)
	notEligible := total - eligibleCount

	// Now differentiate between filtered (annotated with wrong class) and not yet annotated
	var filteredWrongClass, notYetAnnotated int

	// Find task and check if it has dependencies
	stageIndex := a.findTaskIndex(taskID)

	if stageIndex != -1 {
		task := a.Config.Tasks[stageIndex]

		// If task has dependencies, analyze the not-eligible images
		if len(task.If) > 0 {
			// Pre-fetch all dependency data before looping (optimization: move queries outside loop)
			imageHashesByDep, err := a.getDependencyImageHashes(ctx, task)
			if err != nil {
				return nil, err
			}

			// Get all images (using cache)
			allImages, err := a.getCachedImageList(ctx)
			if err != nil {
				return nil, fmt.Errorf("while listing images: %w", err)
			}

			// Get images that passed the filter (eligible)
			eligibleHashes := make(map[string]bool)
			for _, img := range allImages {
				valid := true
				// Check each dependency using pre-fetched map
				for depTaskID := range task.If {
					if !imageHashesByDep[depTaskID][img.SHA256] {
						valid = false
						break
					}
				}

				if valid {
					eligibleHashes[img.SHA256] = true
				}
			}

			// Check not-eligible images to see if they were annotated in dependency phase
			for _, img := range allImages {
				if !eligibleHashes[img.SHA256] {
					// This image is not eligible - check if it was annotated in dependency phase
					annotatedInDep := false
					for depTaskID := range task.If {
						depStageIndex := a.findTaskIndex(depTaskID)
						if depStageIndex == -1 {
							continue
						}

						// Check if this image has ANY annotation in the dependency phase
						hasAnnotation, err := a.annotationRepo.CheckAnnotationExists(ctx, img.SHA256, "", int64(depStageIndex))
						if err == nil && hasAnnotation {
							annotatedInDep = true
							break
						}
					}

					if annotatedInDep {
						filteredWrongClass++
					} else {
						notYetAnnotated++
					}
				}
			}
		} else {
			// No dependencies, so all not-eligible images are "not yet annotated"
			notYetAnnotated = notEligible
		}
	}

	// Calculate percentages
	var completedPercent, pendingPercent, filteredPercent, notYetAnnotatedPercent float64
	if total > 0 {
		completedPercent = float64(completed) / float64(total) * 100
		pendingPercent = float64(pending) / float64(total) * 100
		filteredPercent = float64(filteredWrongClass) / float64(total) * 100
		notYetAnnotatedPercent = float64(notYetAnnotated) / float64(total) * 100
	}

	return &PhaseProgress{
		Completed:              completed,
		Pending:                pending,
		FilteredWrongClass:     filteredWrongClass,
		NotYetAnnotated:        notYetAnnotated,
		Total:                  total,
		CompletedPercent:       completedPercent,
		PendingPercent:         pendingPercent,
		FilteredPercent:        filteredPercent,
		NotYetAnnotatedPercent: notYetAnnotatedPercent,
	}, nil
}

func (a *AnnotatorApp) NextAnnotationStep(ctx context.Context, taskID string) (*AnnotationStep, error) {
	// If no task specified, try each task in order
	if taskID == "" {
		for _, task := range a.Config.Tasks {
			step, err := a.NextAnnotationStep(ctx, task.ID)
			if err != nil {
				return nil, err
			}
			if step == nil {
				continue
			}
			return step, nil
		}
		return nil, nil
	}

	// Find stage index for this task
	stageIndex := a.findTaskIndex(taskID)
	if stageIndex == -1 {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	task := a.Config.Tasks[stageIndex]

	// Pre-fetch all dependency data before looping (optimization: move queries outside loop)
	imageHashesByDep := make(map[string]map[string]bool)
	if len(task.If) > 0 {
		var err error
		imageHashesByDep, err = a.getDependencyImageHashes(ctx, task)
		if err != nil {
			return nil, err
		}
	}

	// Get images without annotation for this stage (using cache)
	allImages, err := a.getCachedImageList(ctx)
	if err != nil {
		return nil, fmt.Errorf("while listing images: %w", err)
	}

	// Filter images based on dependencies and annotation status
	var candidateImages []string
	for _, img := range allImages {
		// Check if image already has annotation for this stage
		hasAnnotation, err := a.annotationRepo.CheckAnnotationExists(ctx, img.SHA256, "", int64(stageIndex))
		if err != nil {
			return nil, err
		}
		if hasAnnotation {
			continue // Skip images that already have annotation
		}

		// Check task dependencies (If field) using pre-fetched map
		valid := true
		if len(task.If) > 0 {
			for depTaskID := range task.If {
				if !imageHashesByDep[depTaskID][img.SHA256] {
					valid = false
					break
				}
			}
		}

		if valid {
			candidateImages = append(candidateImages, img.SHA256)
			// Limit candidates to OffsetAdvance for performance
			if len(candidateImages) >= a.OffsetAdvance {
				break
			}
		}
	}

	// No images available
	if len(candidateImages) == 0 {
		return nil, nil
	}

	// Randomly select one image SHA256
	selectedSHA256 := candidateImages[rand.Intn(len(candidateImages))]

	// Get image details
	selectedImage, err := a.imageRepo.GetBySHA256(ctx, selectedSHA256)
	if err != nil {
		return nil, fmt.Errorf("while getting image details: %w", err)
	}

	return &AnnotationStep{
		TaskID:    taskID,
		ImageID:   selectedSHA256,
		ImageName: selectedImage.Filename,
	}, nil
}

func (a *AnnotatorApp) GetImageFilename(ctx context.Context, sha256 string) (filename string, err error) {
	// Get image from repository using SHA256 hash
	img, err := a.imageRepo.GetBySHA256(ctx, sha256)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("image not found: %s", sha256)
		}
		return "", err
	}

	return img.Filename, nil
}

type AnnotationResponse struct {
	ImageID string
	TaskID  string
	User    string
	Value   string
	Sure    bool
}

func (a *AnnotatorApp) SubmitAnnotation(ctx context.Context, annotation AnnotationResponse) error {
	// Find stage index for this task
	stageIndex := a.findTaskIndex(annotation.TaskID)
	if stageIndex == -1 {
		return fmt.Errorf("no such task: %s", annotation.TaskID)
	}

	// ImageID is already the SHA256 hash, use it directly
	_, err := a.annotationRepo.Create(ctx, annotation.ImageID, annotation.User, stageIndex, annotation.Value)
	if err != nil {
		return fmt.Errorf("while creating annotation: %w", err)
	}

	return nil
}

func (a *AnnotatorApp) GetTask(taskID string) *ConfigTask {
	for _, currentTask := range a.Config.Tasks {
		if currentTask.ID == taskID {
			return currentTask
		}
	}
	return nil
}

// ClassButton represents a class button with keyboard shortcut
type ClassButton struct {
	ID   string
	Name string
	Key  string
}
