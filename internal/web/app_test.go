package web

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestErrTaskNotFoundSentinel(t *testing.T) {
	a := &AnnotatorApp{Config: &Config{}}
	ctx := t.Context()

	_, err := a.CountEligibleImages(ctx, "missing-task")
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("CountEligibleImages: got %v, want ErrTaskNotFound", err)
	}

	_, err = a.CountAvailableImages(ctx, "missing-task")
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("CountAvailableImages: got %v, want ErrTaskNotFound", err)
	}

	_, err = a.NextAnnotationStep(ctx, "missing-task")
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("NextAnnotationStep: got %v, want ErrTaskNotFound", err)
	}

	err = a.SubmitAnnotation(ctx, AnnotationResponse{TaskID: "missing-task"})
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("SubmitAnnotation: got %v, want ErrTaskNotFound", err)
	}

	if got := a.GetTask("missing-task"); got != nil {
		t.Fatalf("GetTask(missing): got %#v, want nil", got)
	}

	_, _, err = a.lookupTask("missing-task")
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("lookupTask: got %v, want ErrTaskNotFound", err)
	}
}

func TestLookupTaskFindsConfiguredTask(t *testing.T) {
	task := &ConfigTask{ID: "quality"}
	a := &AnnotatorApp{Config: &Config{Tasks: []*ConfigTask{task}}}

	got, idx, err := a.lookupTask("quality")
	if err != nil {
		t.Fatalf("lookupTask: %v", err)
	}
	if idx != 0 || got != task {
		t.Fatalf("lookupTask: got task=%#v idx=%d, want task=%#v idx=0", got, idx, task)
	}
	if a.GetTask("quality") != task {
		t.Fatalf("GetTask: got %#v, want %#v", a.GetTask("quality"), task)
	}
}

func TestSecureJoinPathTraversalSentinel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "securejoin_sentinel")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("cleanup: %v", err)
		}
	})

	absBase, err := filepath.Abs(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = secureJoin(absBase, "../secret.txt")
	if !errors.Is(err, ErrPathTraversal) {
		t.Fatalf("secureJoin: got %v, want ErrPathTraversal", err)
	}
}
