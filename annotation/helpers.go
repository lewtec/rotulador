package annotation

import (
	"context"
	"fmt"
)

// findTaskIndex finds the index of a task by its ID
func (a *AnnotatorApp) findTaskIndex(taskID string) int {
	for i, task := range a.Config.Tasks {
		if task.ID == taskID {
			return i
		}
	}
	return -1
}

// getDependencyImageHashes fetches the image hashes for all dependencies of a task.
// It returns a map where keys are dependency task IDs and values are sets of valid image hashes (SHA256).
// This allows for O(1) lookup to check if an image satisfies a dependency.
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
			return nil, fmt.Errorf("while checking dependency for task %s (requires %s=%s): %w", task.ID, depTaskID, requiredValue, err)
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
