package annotation

import (
	"embed"
	"html/template"
	"io"
	"io/fs"
	"sync"

	"github.com/abiosoft/mold"
)

// TemplateManager manages templates using mold for layout inheritance
type TemplateManager struct {
	engine mold.Engine
	mu     sync.RWMutex
}

// BlockData represents data that can be passed to templates
type BlockData struct {
	Title   string
	Content any
	Data    any
	Blocks  map[string]any
}

// NewTemplateManager creates a new template manager using mold.
// The fs should be an embed.FS containing your templates under "templates/"
// with layout "layout.html".
func NewTemplateManager(templateFS embed.FS, options ...mold.Option) (*TemplateManager, error) {
	return newTemplateManager(templateFS, withDefaultLayout(options)...)
}

// NewTemplateManagerWithFuncMap creates a template manager with custom template
// functions. Same defaults as NewTemplateManager.
func NewTemplateManagerWithFuncMap(templateFS embed.FS, funcMap template.FuncMap, options ...mold.Option) (*TemplateManager, error) {
	opts := make([]mold.Option, 0, len(options)+1)
	opts = append(opts, options...)
	opts = append(opts, mold.WithFuncMap(funcMap))
	return NewTemplateManager(templateFS, opts...)
}

// NewTemplateManagerWithFS creates a template manager from a plain fs.FS
// without the embed root/layout defaults.
func NewTemplateManagerWithFS(fsys fs.FS, options ...mold.Option) (*TemplateManager, error) {
	return newTemplateManager(fsys, options...)
}

func withDefaultLayout(options []mold.Option) []mold.Option {
	opts := make([]mold.Option, 0, len(options)+2)
	opts = append(opts, options...)
	opts = append(opts, mold.WithRoot("templates"), mold.WithLayout("layout.html"))
	return opts
}

func newTemplateManager(fsys fs.FS, options ...mold.Option) (*TemplateManager, error) {
	engine, err := mold.New(fsys, options...)
	if err != nil {
		return nil, err
	}
	return &TemplateManager{engine: engine}, nil
}

// Render renders a page template (mold will automatically handle layout inheritance)
func (tm *TemplateManager) Render(w io.Writer, pageName string, data any) error {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.engine.Render(w, pageName, data)
}

// RenderWithBlocks renders a template with explicit block definitions
func (tm *TemplateManager) RenderWithBlocks(w io.Writer, templateName string, blocks map[string]any) error {
	return tm.Render(w, templateName, blocks)
}

// AddFuncMap adds custom template functions
func (tm *TemplateManager) AddFuncMap(funcMap template.FuncMap) {
	// Note: With the new mold API, functions should be added during creation using WithFuncMap
	// This method is kept for backwards compatibility but won't work with an already-created engine
	// Consider recreating the engine with WithFuncMap option instead
}
