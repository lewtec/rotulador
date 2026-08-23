package web

import "testing"

func TestImageMeetsDependencies(t *testing.T) {
	t.Parallel()

	hashes := map[string]map[string]bool{
		"quality": {"aaa": true, "ccc": true},
		"person":  {"aaa": true, "bbb": true},
	}

	tests := []struct {
		name     string
		sha      string
		required map[string]string
		want     bool
	}{
		{name: "no dependencies", sha: "zzz", required: nil, want: true},
		{name: "empty dependencies", sha: "zzz", required: map[string]string{}, want: true},
		{name: "single dep match", sha: "aaa", required: map[string]string{"quality": "good"}, want: true},
		{name: "single dep miss", sha: "bbb", required: map[string]string{"quality": "good"}, want: false},
		{name: "all deps match", sha: "aaa", required: map[string]string{"quality": "good", "person": "true"}, want: true},
		{name: "partial dep miss", sha: "bbb", required: map[string]string{"quality": "good", "person": "true"}, want: false},
		{name: "unknown dep key", sha: "aaa", required: map[string]string{"missing": "x"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := imageMeetsDependencies(tt.sha, tt.required, hashes)
			if got != tt.want {
				t.Fatalf("imageMeetsDependencies(%q) = %v, want %v", tt.sha, got, tt.want)
			}
		})
	}
}
