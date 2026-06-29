package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
)

type Handlers struct {
	db        *sql.DB
	templates *template.Template
}

func NewHandlers(db *sql.DB) Handlers {
	return Handlers{
		db:        db,
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
