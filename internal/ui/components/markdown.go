package components

import (
	"context"
	"io"

	"github.com/a-h/templ"
	"github.com/russross/blackfriday/v2"
)

// Markdown renders trusted markdown (config/YAML) as HTML via blackfriday.
func Markdown(text string) templ.Component {
	html := string(blackfriday.Run([]byte(text)))
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, html)
		return err
	})
}
