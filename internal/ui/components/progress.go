package components

// Progress is the segmented phase progress bar model.
type Progress struct {
	Completed              int
	Pending                int
	FilteredWrongClass     int
	NotYetAnnotated        int
	Total                  int
	CompletedPercent       float64
	PendingPercent         float64
	FilteredPercent        float64
	NotYetAnnotatedPercent float64
}
