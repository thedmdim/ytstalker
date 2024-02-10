package handlers

import (
	"log"
	"net/http"

	"html/template"

	"github.com/gorilla/mux"
	"zombiezen.com/go/sqlite/sqlitex"
)

var templates = template.Must(template.ParseGlob("web/*/*.html"))

type Router struct {
	mux.Router
	db  *sqlitex.Pool
}

func NewRouter(db *sqlitex.Pool) *Router {

	router := &Router{
		Router: *mux.NewRouter(),
		db:     db,
	}

	// api
	router.PathPrefix("/rating").Methods("GET").HandlerFunc(router.GetRating)
	router.PathPrefix("/random").Methods("GET").HandlerFunc(router.RedirectRandom)

	router.PathPrefix("/{cam_id}/{reaction:(?:like|dislike)}").Methods("POST").HandlerFunc(router.WriteReaction).HeadersRegexp("visitor", "[0-9]{10,20}")
	router.PathPrefix("/{cam_id}/stream").Methods("GET").HandlerFunc(router.ProxyStream)
	router.PathPrefix("/{cam_id}").Methods("GET").HandlerFunc(router.GetCam)
	router.PathPrefix("/").Methods("GET").HandlerFunc(router.RedirectRandom)
	// serve static
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	router.Use(loggingMiddleware)

	return router
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do stuff here
		log.Println(r.RequestURI)
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}
