package web

import (
	"context"
	"fmt"
	"strings"

	"github.com/lewtec/rotulador/internal/domain"
)

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

// findTaskIndex returns the index of the task with the given ID, or -1 if not found.
func (a *AnnotatorApp) findTaskIndex(taskID string) int {
	for i, task := range a.Config.Tasks {
		if task.ID == taskID {
			return i
		}
	}
	return -1
}

// lookupTask returns the task and its stage index, or ErrTaskNotFound.
func (a *AnnotatorApp) lookupTask(taskID string) (*ConfigTask, int, error) {
	i := a.findTaskIndex(taskID)
	if i == -1 {
		return nil, -1, fmt.Errorf("%w: %s", ErrTaskNotFound, taskID)
	}
	return a.Config.Tasks[i], i, nil
}

// imageMeetsDependencies reports whether sha256 is present in every
// required dependency hash set. An empty required map is always true.
func imageMeetsDependencies(sha256 string, required map[string]string, hashes map[string]map[string]bool) bool {
	for depTaskID := range required {
		if !hashes[depTaskID][sha256] {
			return false
		}
	}
	return true
}

// listImagesForTask returns the cached image list and, when the task has
// dependencies, the pre-fetched hash sets used by imageMeetsDependencies.
func (a *AnnotatorApp) listImagesForTask(ctx context.Context, task *ConfigTask) ([]*domain.Image, map[string]map[string]bool, error) {
	var hashes map[string]map[string]bool
	if len(task.If) > 0 {
		var err error
		hashes, err = a.getDependencyImageHashes(ctx, task)
		if err != nil {
			return nil, nil, err
		}
	}
	images, err := a.getCachedImageList(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("while listing images: %w", err)
	}
	return images, hashes, nil
}

// imagesMeetingDependencies keeps images present in every required dependency set.
// An empty required map returns images unchanged.
func imagesMeetingDependencies(images []*domain.Image, required map[string]string, hashes map[string]map[string]bool) []*domain.Image {
	if len(required) == 0 {
		return images
	}
	out := make([]*domain.Image, 0, len(images))
	for _, img := range images {
		if imageMeetsDependencies(img.SHA256, required, hashes) {
			out = append(out, img)
		}
	}
	return out
}

// pendingImages keeps images for which exists is false.
// limit <= 0 means no cap.
func pendingImages(images []*domain.Image, exists func(sha256 string) (bool, error), limit int) ([]*domain.Image, error) {
	var out []*domain.Image
	for _, img := range images {
		has, err := exists(img.SHA256)
		if err != nil {
			return nil, err
		}
		if has {
			continue
		}
		out = append(out, img)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// pendingImagesForTask returns images that meet task.If and have no annotation
// at stageIndex. limit <= 0 means no cap.
func (a *AnnotatorApp) pendingImagesForTask(ctx context.Context, task *ConfigTask, stageIndex int, limit int) ([]*domain.Image, error) {
	allImages, hashes, err := a.listImagesForTask(ctx, task)
	if err != nil {
		return nil, err
	}
	return pendingImages(imagesMeetingDependencies(allImages, task.If, hashes), func(sha256 string) (bool, error) {
		return a.annotationRepo.CheckAnnotationExists(ctx, sha256, "", int64(stageIndex))
	}, limit)
}

// getDependencyImageHashes pre-fetches image hashes for all dependencies of the given task.
// This optimization moves queries outside the main loop.
func (a *AnnotatorApp) getDependencyImageHashes(ctx context.Context, task *ConfigTask) (map[string]map[string]bool, error) {
	imageHashesByDep := make(map[string]map[string]bool)
	for depTaskID, requiredValue := range task.If {
		// Find the stage index for the dependency task
		depStageIndex := a.findTaskIndex(depTaskID)
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
	return imageHashesByDep, nil
}
