package annotation

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/lewtec/rotulador/internal/repository"
)

type AnnotatorApp struct {
	ImagesDir      string
	Database       *sql.DB
	Config         *Config
	Logger         *slog.Logger
	OffsetAdvance  int
	imageRepo      *repository.ImageRepository
	annotationRepo *repository.AnnotationRepository
}

func (a *AnnotatorApp) init() {
	if a.ImagesDir[len(a.ImagesDir)-1] == '/' {
		a.ImagesDir = a.ImagesDir[:len(a.ImagesDir)-1]
	}
	if a.OffsetAdvance == 0 {
		a.OffsetAdvance = 10
	}
	// Initialize repositories
	a.imageRepo = repository.NewImageRepository(a.Database)
	a.annotationRepo = repository.NewAnnotationRepository(a.Database)
}

func (a *AnnotatorApp) GetHTTPHandler() http.Handler {
	a.init()
	mux := http.NewServeMux()

	// Home page
	mux.HandleFunc("/", a.HandleHome)

	// Favicon handler
	mux.HandleFunc("/favicon.svg", a.HandleFavicon)

	// Help pages
	mux.HandleFunc("/help/", a.HandleHelp)

	// Annotate pages
	mux.HandleFunc("/annotate/", a.HandleAnnotate)

	// Asset handler - serves images by SHA256 hash
	mux.HandleFunc("/asset/", a.HandleAsset)

	a.Logger.Debug("images dir", "dir", a.ImagesDir)

	var handler http.Handler = mux
	loggerMiddleware := NewHTTPLogger(a.Logger)
	handler = i18nMiddleware(handler)
	handler = loggerMiddleware.Middleware(handler)
	handler = a.authenticationMiddleware(handler)
	handler = requestCacheMiddleware(handler)
	return handler
}
