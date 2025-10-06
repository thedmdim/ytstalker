package handlers

import (
	"log"
	"net/http"

	"html/template"

	"zombiezen.com/go/sqlite/sqlitex"
)

// shared to all handlers field
type Handlers struct {
	db *sqlitex.Pool
	templates *template.Template
}

func NewHandlers(db *sqlitex.Pool) Handlers {
	return Handlers{
		db: db,
		templates: template.Must(template.ParseGlob("web/*/*.html")),
	}
}

func (h Handlers) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do stuff here
		log.Println(r.RequestURI)
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

func (h Handlers) CacheHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "max-age=3600")
			next.ServeHTTP(w, r)
	})
}
