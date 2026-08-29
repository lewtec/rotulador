package web

import (
	"errors"
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

func TestPendingImages(t *testing.T) {
	t.Parallel()

	aaa := &domain.Image{SHA256: "aaa"}
	bbb := &domain.Image{SHA256: "bbb"}
	ccc := &domain.Image{SHA256: "ccc"}
	images := []*domain.Image{aaa, bbb, ccc}
	annotated := map[string]bool{"bbb": true}

	exists := func(sha256 string) (bool, error) {
		return annotated[sha256], nil
	}

	t.Run("keeps unannotated in order", func(t *testing.T) {
		t.Parallel()
		got, err := pendingImages(images, exists, 0)
		if err != nil {
			t.Fatalf("pendingImages: %v", err)
		}
		if len(got) != 2 || got[0] != aaa || got[1] != ccc {
			t.Fatalf("got %#v, want [aaa ccc]", got)
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		t.Parallel()
		got, err := pendingImages(images, exists, 1)
		if err != nil {
			t.Fatalf("pendingImages: %v", err)
		}
		if len(got) != 1 || got[0] != aaa {
			t.Fatalf("got %#v, want [aaa]", got)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		got, err := pendingImages(nil, exists, 0)
		if err != nil {
			t.Fatalf("pendingImages: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("got %#v, want empty", got)
		}
	})

	t.Run("exists error", func(t *testing.T) {
		t.Parallel()
		_, err := pendingImages(images, func(string) (bool, error) {
			return false, errPendingProbe
		}, 0)
		if !errors.Is(err, errPendingProbe) {
			t.Fatalf("got %v, want %v", err, errPendingProbe)
		}
	})
}

var errPendingProbe = errors.New("probe failed")
