package web

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lewtec/rotulador/internal/ui/pages"
)

// TestAnnotateTemplateKeepsFilenameOutOfJSString ensures image filenames are
// not interpolated into a JS string literal inside an HTML attribute.
func TestAnnotateTemplateKeepsFilenameOutOfJSString(t *testing.T) {
	const evil = `x');alert(1);//`

	var buf bytes.Buffer
	comp := pages.Annotate(PageShell("Annotate"), pages.AnnotateData{
		TaskID:        "phase1",
		TaskName:      "Phase 1",
		ImageID:       "abc123",
		ImageFilename: evil,
		Classes:       []pages.ClassButton{},
	})
	if err := comp.Render(t.Context(), &buf); err != nil {
		t.Fatalf("Annotate.Render: %v", err)
	}
	out := buf.String()

	if strings.Contains(out, "writeText('") || strings.Contains(out, `writeText("`) {
		t.Fatalf("filename still interpolated into writeText string literal:\n%s", out)
	}
	if !strings.Contains(out, `data-filename=`) {
		t.Fatalf("expected data-filename attribute, got:\n%s", out)
	}
	if !strings.Contains(out, "this.dataset.filename") {
		t.Fatalf("expected this.dataset.filename usage, got:\n%s", out)
	}
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
