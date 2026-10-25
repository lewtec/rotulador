package i18n

import "testing"

func TestTKnownAndMissing(t *testing.T) {
	if got := T(t.Context(), "Home"); got != "Home" {
		t.Fatalf("T(Home) = %q", got)
	}
	const missing = "message-that-does-not-exist"
	if got := T(t.Context(), missing); got != missing {
		t.Fatalf("T(missing) = %q", got)
	}
}

func TestTDataTemplateAndMissing(t *testing.T) {
	const id = "janitor-hello-name"
	if err := AddMessage("en", id, "Hello {{.Name}}"); err != nil {
		t.Fatalf("AddMessage: %v", err)
	}
	got := TData(t.Context(), id, map[string]any{"Name": "Ada"})
	if got != "Hello Ada" {
		t.Fatalf("TData = %q", got)
	}
	const missing = "janitor-missing-template"
	got = TData(t.Context(), missing, map[string]any{"Name": "Ada"})
	if got != missing {
		t.Fatalf("TData(missing) = %q", got)
	}
}
