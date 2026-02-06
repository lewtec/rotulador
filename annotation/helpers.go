package annotation

import (
	"context"
	"fmt"
	"strings"
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
