package annotation

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusCodeRecorder_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	w := NewStatusCodeRecorderResponseWriter(rec)
	w.WriteHeader(http.StatusNotFound)
	if w.Status != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Status)
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("recorder code = %d, want 404", rec.Code)
	}
}

func TestStatusCodeRecorder_WriteImplies200(t *testing.T) {
	rec := httptest.NewRecorder()
	w := NewStatusCodeRecorderResponseWriter(rec)
	if _, err := w.Write([]byte("ok")); err != nil {
		t.Fatal(err)
	}
	if w.Status != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Status)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("recorder code = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestStatusCodeRecorder_WriteAfterHeaderKeepsStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	w := NewStatusCodeRecorderResponseWriter(rec)
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write([]byte("created")); err != nil {
		t.Fatal(err)
	}
	if w.Status != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Status)
	}
}

func TestStatusCodeRecorder_Unwrap(t *testing.T) {
	rec := httptest.NewRecorder()
	w := NewStatusCodeRecorderResponseWriter(rec)
	if w.Unwrap() != rec {
		t.Fatal("Unwrap did not return underlying writer")
	}
}
