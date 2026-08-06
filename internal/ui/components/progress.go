package components

// ProgressSegment is one colored slice of the phase progress bar.
type ProgressSegment struct {
	Count   int
	Percent float64
	// Label is an i18n message id used in the segment title tooltip.
	Label string
	// Class is the Tailwind/daisyUI class for the segment fill (e.g. "bg-success ...").
	Class string
}

// Progress is a segmented progress bar: zero or more segments summing toward Total.
type Progress struct {
	Total    int
	Segments []ProgressSegment
}
