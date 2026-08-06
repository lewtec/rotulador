package pages

import "github.com/lewtec/rotulador/internal/ui/components"

type HomeData struct {
	Description string
}

type ClassButton struct {
	ID   string
	Name string // i18n message id
	Key  string
}

type AnnotateProgress struct {
	CompletedCount int
	TotalCount     int
}

type AnnotateData struct {
	TaskID        string
	TaskName      string
	ImageID       string
	ImageFilename string
	Classes       []ClassButton
	PhaseProgress *components.Progress
	Progress      *AnnotateProgress
}

type HelpClass struct {
	ID          string
	Name        string // i18n message id
	Description string
	Examples    []string
}

type HelpTask struct {
	ID             string
	Name           string
	ShortName      string
	AvailableCount int
	TotalCount     int
	CompletedCount int
	PhaseProgress  *components.Progress
	If             map[string]string
	Classes        []HelpClass
}

type HelpData struct {
	Description string
	// Detail is set for /help/{task}; Tasks holds timeline and single-task progress card.
	Detail *HelpTask
	Tasks  []HelpTask
}
