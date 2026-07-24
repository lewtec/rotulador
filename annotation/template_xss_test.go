package annotation

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// TestAnnotateTemplateKeepsFilenameOutOfJSString ensures image filenames are
// not interpolated into a JS string literal inside an HTML attribute.
// html/template HTML-escapes quotes, but the browser decodes entities before
// running event-handler scripts, so a crafted name can break out of quotes.
func TestAnnotateTemplateKeepsFilenameOutOfJSString(t *testing.T) {
	const evil = `x');alert(1);//`

	var buf bytes.Buffer
	err := RenderPageWithContext(context.Background(), &buf, "annotate.html", map[string]any{
		"Title":         "Annotate",
		"TaskID":        "phase1",
		"TaskName":      "Phase 1",
		"ImageID":       "abc123",
		"ImageFilename": evil,
		"Classes":       []ClassButton{},
	})
	if err != nil {
		t.Fatalf("RenderPageWithContext: %v", err)
	}
	out := buf.String()

	// Must not embed the raw quote-break sequence inside writeText('…').
	if strings.Contains(out, "writeText('") || strings.Contains(out, `writeText("`) {
		t.Fatalf("filename still interpolated into writeText string literal:\n%s", out)
	}
	// Safe pattern: HTML-escaped data attribute + dataset read.
	if !strings.Contains(out, `data-filename=`) {
		t.Fatalf("expected data-filename attribute, got:\n%s", out)
	}
	if !strings.Contains(out, "this.dataset.filename") {
		t.Fatalf("expected this.dataset.filename usage, got:\n%s", out)
	}
	// After simulating HTML entity decode of the full document, the old
	// breakout must not appear as executable JS source next to writeText.
	decoded := out
	for _, pair := range []struct{ old, new string }{
		{"&#39;", "'"},
		{"&#34;", `"`},
		{"&quot;", `"`},
		{"&lt;", "<"},
		{"&gt;", ">"},
		{"&amp;", "&"},
	} {
		decoded = strings.ReplaceAll(decoded, pair.old, pair.new)
	}
	if strings.Contains(decoded, "writeText('x');alert(1)") {
		t.Fatalf("decoded output still allows JS breakout via writeText string")
	}
}
