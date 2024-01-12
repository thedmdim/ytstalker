package api

import (
	"log"
	"net/http"
	"ytstalker/app/youtube"

	"html/template"

	"github.com/gorilla/mux"
	"zombiezen.com/go/sqlite/sqlitex"
)

var templates = template.Must(template.ParseGlob("web/*/*.html"))

type Router struct {
	mux.Router
	db  *sqlitex.Pool
	ytr *youtube.YouTubeRequester
}

func NewRouter(db *sqlitex.Pool, ytr *youtube.YouTubeRequester) *Router {

	router := &Router{
		Router: *mux.NewRouter(),
		db:     db,
		ytr:    ytr,
	}

	// api
	router.PathPrefix("/api/videos/random").Methods("GET").HandlerFunc(router.GetRandom).HeadersRegexp("visitor", "[0-9]{10,20}")
	router.PathPrefix("/api/videos/{video_id}/{reaction:(?:cool|trash)}").Methods("POST").HandlerFunc(router.WriteReaction).HeadersRegexp("visitor", "[0-9]{10,20}")
	router.PathPrefix("/api/videos/{video_id}").Methods("GET").HandlerFunc(router.GetVideo)

	// pages
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web`/static"))))
	router.PathPrefix("/").Methods("GET").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			err := templates.ExecuteTemplate(w, "random.html", nil)
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
