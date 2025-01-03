package handlers

import (
	"log"
	"net/http"

	"html/template"

	"github.com/gorilla/mux"
	"zombiezen.com/go/sqlite/sqlitex"
)

var Templates *template.Template

type Router struct {
	mux.Router
	db *sqlitex.Pool
}

func NewRouter(db *sqlitex.Pool) *Router {

	router := &Router{
		Router: *mux.NewRouter(),
		db:     db,
	}

	// api
	router.PathPrefix("/api/videos/random").Methods("GET").HandlerFunc(router.GetRandom).HeadersRegexp("visitor", "[0-9]{10,20}")
	router.PathPrefix("/api/videos/{video_id}/{reaction:(?:cool|trash)}").Methods("POST").HandlerFunc(router.WriteReaction).HeadersRegexp("visitor", "[0-9]{10,20}")
	router.PathPrefix("/api/videos/{video_id}").Methods("GET").HandlerFunc(router.GetVideo)
	// pages
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	router.PathPrefix("/stats").Methods("GET").HandlerFunc(router.GetStats)
	router.PathPrefix("/").Methods("GET").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			err := Templates.ExecuteTemplate(w, "random.html", nil)
			if err != nil {
				log.Println(err.Error())
			}
		})

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
