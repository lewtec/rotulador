package annotation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"math/rand"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/lewtec/rotulador/db/migrations"
	"github.com/lewtec/rotulador/internal/domain"
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

func (a *AnnotatorApp) init() {
	if a.ImagesDir[len(a.ImagesDir)-1] == '/' {
		a.ImagesDir = a.ImagesDir[:len(a.ImagesDir)-1]
	}
	if a.OffsetAdvance == 0 {
		a.OffsetAdvance = 10
	}
	// Initialize repositories
	a.imageRepo = repository.NewImageRepository(a.Database)
	a.annotationRepo = repository.NewAnnotationRepository(a.Database)
}

func pathParts(path string) []string {
	parts := strings.Split(path, "/")
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
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
	stageIndex := -1
	for i, task := range a.Config.Tasks {
		if task.ID == taskID {
			stageIndex = i
			break
		}
	}
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
	imageHashesByDep := make(map[string]map[string]bool)
	for depTaskID, requiredValue := range task.If {
		// Find the stage index for the dependency task
		depStageIndex := -1
		for i, t := range a.Config.Tasks {
			if t.ID == depTaskID {
				depStageIndex = i
				break
			}
		}
		if depStageIndex == -1 {
			continue
		}

		// Fetch all image hashes for this dependency ONCE
		imageHashes, err := a.annotationRepo.GetImageHashesWithAnnotation(ctx, int64(depStageIndex), requiredValue)
		if err != nil {
			return 0, fmt.Errorf("while checking dependency: %w", err)
		}

		// Convert to map for O(1) lookup
		hashSet := make(map[string]bool, len(imageHashes))
		for _, hash := range imageHashes {
			hashSet[hash] = true
		}
		imageHashesByDep[depTaskID] = hashSet
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
	stageIndex := -1
	for i, task := range a.Config.Tasks {
		if task.ID == taskID {
			stageIndex = i
			break
		}
	}
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
		imageHashesByDep := make(map[string]map[string]bool)
		for depTaskID, requiredValue := range task.If {
			// Find the stage index for the dependency task
			depStageIndex := -1
			for i, t := range a.Config.Tasks {
				if t.ID == depTaskID {
					depStageIndex = i
					break
				}
			}
			if depStageIndex == -1 {
				continue
			}

			// Fetch all image hashes for this dependency ONCE
			imageHashes, err := a.annotationRepo.GetImageHashesWithAnnotation(ctx, int64(depStageIndex), requiredValue)
			if err != nil {
				return 0, fmt.Errorf("while checking dependency: %w", err)
			}

			// Convert to map for O(1) lookup
			hashSet := make(map[string]bool, len(imageHashes))
			for _, hash := range imageHashes {
				hashSet[hash] = true
			}
			imageHashesByDep[depTaskID] = hashSet
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
	stageIndex := -1
	for i, task := range a.Config.Tasks {
		if task.ID == taskID {
			stageIndex = i
			break
		}
	}

	if stageIndex != -1 {
		task := a.Config.Tasks[stageIndex]

		// If task has dependencies, analyze the not-eligible images
		if len(task.If) > 0 {
			// Pre-fetch all dependency data before looping (optimization: move queries outside loop)
			imageHashesByDep := make(map[string]map[string]bool)
			for depTaskID, requiredValue := range task.If {
				depStageIndex := -1
				for i, t := range a.Config.Tasks {
					if t.ID == depTaskID {
						depStageIndex = i
						break
					}
				}
				if depStageIndex == -1 {
					continue
				}

				// Fetch all image hashes for this dependency ONCE
				imageHashes, err := a.annotationRepo.GetImageHashesWithAnnotation(ctx, int64(depStageIndex), requiredValue)
				if err != nil {
					return nil, fmt.Errorf("while checking dependency: %w", err)
				}

				// Convert to map for O(1) lookup
				hashSet := make(map[string]bool, len(imageHashes))
				for _, hash := range imageHashes {
					hashSet[hash] = true
				}
				imageHashesByDep[depTaskID] = hashSet
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
						depStageIndex := -1
						for i, t := range a.Config.Tasks {
							if t.ID == depTaskID {
								depStageIndex = i
								break
							}
						}
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
	stageIndex := -1
	for i, task := range a.Config.Tasks {
		if task.ID == taskID {
			stageIndex = i
			break
		}
	}
	if stageIndex == -1 {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	task := a.Config.Tasks[stageIndex]

	// Pre-fetch all dependency data before looping (optimization: move queries outside loop)
	imageHashesByDep := make(map[string]map[string]bool)
	if len(task.If) > 0 {
		for depTaskID, requiredValue := range task.If {
			// Find the stage index for the dependency task
			depStageIndex := -1
			for i, t := range a.Config.Tasks {
				if t.ID == depTaskID {
					depStageIndex = i
					break
				}
			}
			if depStageIndex == -1 {
				continue
			}

			// Fetch all image hashes for this dependency ONCE
			imageHashes, err := a.annotationRepo.GetImageHashesWithAnnotation(ctx, int64(depStageIndex), requiredValue)
			if err != nil {
				return nil, fmt.Errorf("while checking dependency: %w", err)
			}

			// Convert to map for O(1) lookup
			hashSet := make(map[string]bool, len(imageHashes))
			for _, hash := range imageHashes {
				hashSet[hash] = true
			}
			imageHashesByDep[depTaskID] = hashSet
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
	stageIndex := -1
	for i, task := range a.Config.Tasks {
		if task.ID == annotation.TaskID {
			stageIndex = i
			break
		}
	}
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

func (a *AnnotatorApp) GetHTTPHandler() http.Handler {
	a.init()
	mux := http.NewServeMux()

	// Home page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}

		data := map[string]interface{}{
			"Title":       "Welcome to Rotulador",
			"ProjectName": "Welcome to Rotulador",
			"Description": a.Config.Meta.Description,
		}

		err := RenderPageWithRequest(r, w, "home.html", data)
		if err != nil {
			a.Logger.Error("error rendering home template", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	// Favicon handler
	mux.HandleFunc("/favicon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		if _, err := w.Write([]byte(GetFavicon())); err != nil {
			a.Logger.Error("error writing favicon response", "err", err)
		}
	})

	// Help pages
	mux.HandleFunc("/help/", func(w http.ResponseWriter, r *http.Request) {
		itemPath := pathParts(r.URL.Path)
		title := "Help"

		var tasks []TaskWithCount
		var currentTask *ConfigTask = nil

		if len(itemPath) == 1 {
			// Only populate tasks for the timeline view (no markdown for tasks)
			tasks = make([]TaskWithCount, 0, len(a.Config.Tasks))

			for _, task := range a.Config.Tasks {
				availableCount, err := a.CountAvailableImages(r.Context(), task.ID)
				if err != nil {
					a.Logger.Error("error counting available images", "task", task.ID, "err", err)
					availableCount = 0
				}

				totalEligible, err := a.CountEligibleImages(r.Context(), task.ID)
				if err != nil {
					a.Logger.Error("error counting eligible images", "task", task.ID, "err", err)
					totalEligible = availableCount // fallback to available
				}

				completedCount := totalEligible - availableCount
				if completedCount < 0 {
					completedCount = 0
				}

				// Get comprehensive phase progress stats
				phaseProgress, err := a.GetPhaseProgressStats(r.Context(), task.ID)
				if err != nil {
					a.Logger.Error("error getting phase progress", "task", task.ID, "err", err)
					phaseProgress = &PhaseProgress{}
				}

				tasks = append(tasks, TaskWithCount{
					ConfigTask:     task,
					AvailableCount: availableCount,
					TotalCount:     totalEligible,
					CompletedCount: completedCount,
					PhaseProgress:  phaseProgress,
				})
			}
		} else if len(itemPath) == 2 {
			helpTask := itemPath[1]
			task := a.GetTask(helpTask)
			if task == nil {
				http.NotFoundHandler().ServeHTTP(w, r)
				return
			}
			currentTask = task

			// Get progress stats for this specific task
			phaseProgress, err := a.GetPhaseProgressStats(r.Context(), helpTask)
			if err != nil {
				a.Logger.Error("error getting phase progress", "task", helpTask, "err", err)
				phaseProgress = &PhaseProgress{}
			}

			// Get available count to check if there are images to annotate
			availableCount, err := a.CountAvailableImages(r.Context(), helpTask)
			if err != nil {
				a.Logger.Error("error counting available images", "task", helpTask, "err", err)
				availableCount = 0
			}

			tasks = []TaskWithCount{
				{
					ConfigTask:     task,
					AvailableCount: availableCount,
					TotalCount:     phaseProgress.Completed + phaseProgress.Pending,
					CompletedCount: phaseProgress.Completed,
					PhaseProgress:  phaseProgress,
				},
			}
		} else {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}

		data := map[string]interface{}{
			"Title":       title,
			"Description": a.Config.Meta.Description,
			"Task":        currentTask,
			"Tasks":       tasks,
		}

		err := RenderPageWithRequest(r, w, "help.html", data)
		if err != nil {
			a.Logger.Error("error rendering help template", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	// Annotate pages
	mux.HandleFunc("/annotate/", func(w http.ResponseWriter, r *http.Request) {
		itemPath := pathParts(r.URL.Path)

		if len(itemPath) != 3 {
			taskID := r.URL.Query().Get("task")
			step, err := a.NextAnnotationStep(r.Context(), taskID)
			if err != nil {
				a.Logger.Error("error in annotate when getting next step from scratch", "err", err)
				w.WriteHeader(500)
				return
			}
			if step == nil {
				data := map[string]interface{}{
					"Title": "All annotations are done!",
				}
				err := RenderPageWithRequest(r, w, "complete.html", data)
				if err != nil {
					a.Logger.Error("error rendering complete template", "err", err)
				}
				return
			}
			http.Redirect(w, r, fmt.Sprintf("/annotate/%s/%s", step.TaskID, step.ImageID), http.StatusSeeOther)
			return
		}

		taskID := itemPath[1]
		imageID := itemPath[2]
		task := a.GetTask(taskID)
		if task == nil {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}
		imageFilename, _ := a.GetImageFilename(r.Context(), imageID)

		if r.Method == http.MethodPost {
			a.Logger.Debug("POST")
			if err := r.ParseForm(); err != nil {
				a.Logger.Error("error parsing form", "err", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if !r.Form.Has("selectedClass") || !r.Form.Has("sure") {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			selectedClass := r.FormValue("selectedClass")
			_, isClassValid := task.Classes[selectedClass]
			a.Logger.Debug("Selected class", "class", selectedClass, "empty", selectedClass == "", "valid", isClassValid)
			sure := r.FormValue("sure") == "on"
			a.Logger.Debug("Sure", "sure", sure)
			user, _, _ := r.BasicAuth()
			err := a.SubmitAnnotation(r.Context(), AnnotationResponse{
				ImageID: imageID,
				TaskID:  taskID,
				User:    user,
				Value:   selectedClass,
				Sure:    sure,
			})
			if err != nil {
				a.Logger.Error("error while submitting annotation", "err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			step, err := a.NextAnnotationStep(r.Context(), taskID)
			if err != nil {
				a.Logger.Error("error while getting next step", "err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if step == nil {
				step, err = a.NextAnnotationStep(r.Context(), "")
				if err != nil {
					a.Logger.Error("error while getting next step at the end of task", "err", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
			if step == nil {
				w.Header().Add("HX-Redirect", "/")
			} else if step.TaskID != taskID {
				w.Header().Add("HX-Redirect", fmt.Sprintf("/help/%s", step.TaskID))
			} else {
				w.Header().Add("HX-Redirect", fmt.Sprintf("/annotate/%s/%s", taskID, step.ImageID))
			}
			return
		}

		// Build classes with keyboard shortcuts
		classNames := make([]string, 0, len(task.Classes))
		for class := range task.Classes {
			classNames = append(classNames, class)
		}
		sort.Strings(classNames)

		classes := []ClassButton{}
		keyIndex := 1
		for _, className := range classNames {
			classMeta := task.Classes[className]
			key := ""
			if keyIndex <= 9 {
				key = fmt.Sprintf("%d", keyIndex)
				keyIndex++
			}
			classes = append(classes, ClassButton{
				ID:   className,
				Name: i(classMeta.Name),
				Key:  key,
			})
		}

		// Get comprehensive progress information
		phaseProgress, err := a.GetPhaseProgressStats(r.Context(), taskID)
		if err != nil {
			a.Logger.Error("error getting phase progress", "err", err)
			// Fallback to empty progress
			phaseProgress = &PhaseProgress{}
		}

		data := map[string]interface{}{
			"Title":         "annotation",
			"TaskID":        taskID,
			"TaskName":      task.Name,
			"ImageID":       imageID,
			"ImageFilename": imageFilename,
			"Classes":       classes,
			"PhaseProgress": phaseProgress,
			// Keep old Progress for backward compatibility
			"Progress": map[string]interface{}{
				"AvailableCount": phaseProgress.Pending,
				"TotalCount":     phaseProgress.Completed + phaseProgress.Pending,
				"CompletedCount": phaseProgress.Completed,
			},
		}

		err = RenderPageWithRequest(r, w, "annotate.html", data)
		if err != nil {
			a.Logger.Error("error rendering annotate template", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	// Asset handler - serves images by SHA256 hash
	mux.HandleFunc("/asset/", func(w http.ResponseWriter, r *http.Request) {
		itemPath := pathParts(r.URL.Path)
		if len(itemPath) != 2 {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}
		sha256 := itemPath[1]
		a.Logger.Debug("http: fetching asset", "sha256", sha256)

		// Get image filename from repository
		filename, err := a.GetImageFilename(r.Context(), sha256)
		if err != nil {
			a.Logger.Warn("http: asset was not found", "sha256", sha256, "err", err)
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}

		a.Logger.Debug("http: asset is", "sha256", sha256, "filename", filename)
		fullPath := path.Join(a.ImagesDir, filename)
		f, err := os.Open(fullPath)
		if errors.Is(err, os.ErrNotExist) {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			a.Logger.Error("error: http: while serving image asset", "err", err)
			return
		}
		defer func() {
			if err := f.Close(); err != nil {
				a.Logger.Error("error closing file", "err", err)
			}
		}()
		if _, err := io.Copy(w, f); err != nil {
			a.Logger.Error("error copying file to response", "err", err)
		}
	})

	a.Logger.Debug("images dir", "dir", a.ImagesDir)

	var handler http.Handler = mux
	loggerMiddleware := NewHTTPLogger(a.Logger)
	handler = i18nMiddleware(handler)
	handler = loggerMiddleware.Middleware(handler)
	handler = a.authenticationMiddleware(handler)
	handler = requestCacheMiddleware(handler)
	return handler
}

func (a *AnnotatorApp) authenticationMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			var item *ConfigAuth
			item, ok = a.Config.Authentication[username]
			if ok {
				// SECURITY: Use bcrypt to compare the provided password with the stored hash.
				if CheckPasswordHash(password, item.Password) {
					a.Logger.Info("auth for user: success", "username", username)
					handler.ServeHTTP(w, r)
					return
				}
				a.Logger.Warn("auth for user: bad password", "username", username)
			} else {
				a.Logger.Warn("auth for user: no such user", "username", username)
			}
		} else {
			a.Logger.Warn("auth: no credentials provided")
		}
		a.Logger.Warn("auth: not ok")
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

// PrepareDatabase runs both database migrations and image ingestion synchronously.
// For better startup performance, consider using PrepareDatabaseMigrations() synchronously
// and IngestImages() asynchronously instead.
func (a *AnnotatorApp) PrepareDatabase(ctx context.Context) error {
	if err := a.PrepareDatabaseMigrations(ctx); err != nil {
		return err
	}
	if err := a.IngestImages(ctx); err != nil {
		return err
	}
	a.Logger.Info("PrepareDatabase: success! Database is ready")
	return nil
}

// PrepareDatabaseMigrations runs database schema migrations.
// This must be called synchronously before starting the HTTP server.
func (a *AnnotatorApp) PrepareDatabaseMigrations(ctx context.Context) error {
	a.init()
	db, err := sqlite.WithInstance(a.Database, &sqlite.Config{})
	if err != nil {
		return err
	}
	migrationsFS, err := iofs.New(migrations.Migrations, ".")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", migrationsFS, "sqlite", db)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	a.Logger.Info("PrepareDatabaseMigrations: migrations completed successfully")
	return nil
}

// IngestImages scans the images directory and loads all images into the database.
// This can be called asynchronously after the HTTP server starts.
func (a *AnnotatorApp) IngestImages(ctx context.Context) error {
	a.Logger.Info("IngestImages: starting image ingestion from directory", "dir", a.ImagesDir)

	err := filepath.WalkDir(a.ImagesDir, func(fullPath string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if fullPath == a.ImagesDir {
			return nil
		}
		if info.IsDir() {
			return fmt.Errorf("while checking if item '%s' is a file: datasets must be organized in a flat folder structure; hint: use the 'ingest' subcommand", fullPath)
		}

		a.Logger.Debug("IngestImages: processing image", "path", fullPath)

		// Verify it's an image
		_, err = DecodeImage(fullPath)
		if err != nil {
			return fmt.Errorf("while checking if item '%s' is a file: datasets must be organized in a flat folder structure; hint: use the 'ingest' subcommand", fullPath)
		}

		// Hash the file to get SHA256
		fileHash, err := HashFile(fullPath)
		if err != nil {
			return fmt.Errorf("while hashing image '%s': %w", fullPath, err)
		}

		// Use repository to create image (with upsert behavior via ON CONFLICT)
		_, err = a.imageRepo.Create(ctx, fileHash, info.Name())
		if err != nil {
			// Ignore duplicate errors (hash already exists)
			if !strings.Contains(err.Error(), "UNIQUE constraint") {
				return fmt.Errorf("while inserting image '%s': %w", fullPath, err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("while ingesting images: %w", err)
	}

	a.Logger.Info("IngestImages: completed successfully!")
	return nil
}
