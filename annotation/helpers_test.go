package annotation

import (
	"context"
	"strings"
	"testing"
)

func TestGetDependencyImageHashes_UnknownDep(t *testing.T) {
	app := &AnnotatorApp{
		Config: &Config{
			Tasks: []*ConfigTask{
				{
					ID: "person_age",
					If: map[string]string{"contains_person": "true"},
				},
			},
		},
	}

	_, err := app.getDependencyImageHashes(context.Background(), app.Config.Tasks[0])
	if err == nil {
		t.Fatal("expected error for unknown dependency task id")
	}
	if !strings.Contains(err.Error(), "unknown task") {
		t.Fatalf("error = %v, want mention of unknown task", err)
	}
	if !strings.Contains(err.Error(), "contains_person") {
		t.Fatalf("error = %v, want dependency id contains_person", err)
	}
	if !strings.Contains(err.Error(), "person_age") {
		t.Fatalf("error = %v, want task id person_age", err)
	}
}

func TestGetDependencyImageHashes_NoDeps(t *testing.T) {
	app := &AnnotatorApp{
		Config: &Config{
			Tasks: []*ConfigTask{{ID: "quality"}},
		},
	}
	got, err := app.getDependencyImageHashes(context.Background(), app.Config.Tasks[0])
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}
}

func TestFindTaskIndex(t *testing.T) {
	app := &AnnotatorApp{
		Config: &Config{
			Tasks: []*ConfigTask{
				{ID: "a"},
				{ID: "b"},
			},
		},
	}
	if got := app.findTaskIndex("b"); got != 1 {
		t.Fatalf("findTaskIndex(b) = %d, want 1", got)
	}
	if got := app.findTaskIndex("missing"); got != -1 {
		t.Fatalf("findTaskIndex(missing) = %d, want -1", got)
	}
}

func TestPathParts(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"/annotate/t/i", []string{"annotate", "t", "i"}},
		{"annotate/t/i", []string{"annotate", "t", "i"}},
		{"/help/", []string{"help"}},
		{"/", []string{}},
		{"", []string{}},
	}
	for _, tt := range tests {
		got := pathParts(tt.in)
		if len(got) != len(tt.want) {
			t.Fatalf("pathParts(%q) = %v, want %v", tt.in, got, tt.want)
		}
		for i := range tt.want {
			if got[i] != tt.want[i] {
				t.Fatalf("pathParts(%q) = %v, want %v", tt.in, got, tt.want)
			}
		}
	}
}
