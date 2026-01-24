package annotation

import (
	"context"
	"fmt"
)

// findTaskIndex returns the index of the task with the given ID in the config.
// Returns -1 if not found.
func (a *AnnotatorApp) findTaskIndex(taskID string) int {
	for i, task := range a.Config.Tasks {
		if task.ID == taskID {
			return i
		}
	}
	return -1
}

// getDependencyImageHashes returns a map of dependency task ID to a set of valid image hashes.
// This pre-fetches all required image hashes for the task's dependencies.
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
