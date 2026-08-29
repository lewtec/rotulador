package web

import (
	"testing"

	"github.com/lewtec/rotulador/internal/domain"
)

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

func TestImagesMeetingDependencies(t *testing.T) {
	t.Parallel()

	aaa := &domain.Image{SHA256: "aaa"}
	bbb := &domain.Image{SHA256: "bbb"}
	ccc := &domain.Image{SHA256: "ccc"}
	images := []*domain.Image{aaa, bbb, ccc}
	hashes := map[string]map[string]bool{
		"quality": {"aaa": true, "ccc": true},
		"person":  {"aaa": true, "bbb": true},
	}

	t.Run("no dependencies returns input", func(t *testing.T) {
		t.Parallel()
		got := imagesMeetingDependencies(images, nil, hashes)
		if len(got) != 3 || got[0] != aaa || got[2] != ccc {
			t.Fatalf("got %#v, want original slice", got)
		}
	})

	t.Run("empty dependencies returns input", func(t *testing.T) {
		t.Parallel()
		got := imagesMeetingDependencies(images, map[string]string{}, hashes)
		if len(got) != 3 {
			t.Fatalf("got %d images, want 3", len(got))
		}
	})

	t.Run("filters and keeps order", func(t *testing.T) {
		t.Parallel()
		got := imagesMeetingDependencies(images, map[string]string{"quality": "good"}, hashes)
		if len(got) != 2 || got[0] != aaa || got[1] != ccc {
			t.Fatalf("got %#v, want [aaa ccc]", got)
		}
	})

	t.Run("requires every dependency", func(t *testing.T) {
		t.Parallel()
		got := imagesMeetingDependencies(images, map[string]string{"quality": "good", "person": "true"}, hashes)
		if len(got) != 1 || got[0] != aaa {
			t.Fatalf("got %#v, want [aaa]", got)
		}
	})
}
