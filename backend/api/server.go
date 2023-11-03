package api

import (
	"context"
	"log"
	"net/http"
	"os"
	"ytstalker/backend/conf"
	"ytstalker/backend/youtube"

	"html/template"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/acme/autocert"
	"zombiezen.com/go/sqlite/sqlitex"
)

var templates = template.Must(template.ParseGlob("frontend/*/*.html"))

type Server struct {
	domain string
	addr string
	mux http.Handler
	db  *sqlitex.Pool
	ytr *youtube.YouTubeRequester
}

func NewServer(config *conf.Config) *Server {

	// init db
	db, err := sqlitex.Open(config.DSN, 0, 100)
	if err != nil {
		log.Fatal("cannot open db", err)
	}
	dbScheme, err := os.ReadFile("db.sql")
	if err != nil {
		log.Fatal("cannot open db.sql: ", err.Error())
	}
	conn := db.Get(context.Background())
	if err := sqlitex.ExecuteScript(conn, string(dbScheme), nil); err != nil {
		log.Fatal("cannot create db: ", err)
	}
	db.Put(conn)

	log.Println("database ready")

	// init youtube requester
	ytr := youtube.NewYouTubeRequester(config)

	// init server
	r := mux.NewRouter()

	server := &Server{
		addr: config.Addr,
		domain: config.Domain,
		mux: r,
		db:  db,
		ytr: ytr,
	}

	r.PathPrefix("/api/videos/random").Methods("GET").HandlerFunc(server.GetRandom).HeadersRegexp("visitor", "[0-9]{10,20}")
	r.PathPrefix("/api/videos/{video_id}/{reaction:(?:cool|trash)}").Methods("POST").HandlerFunc(server.WriteReaction).HeadersRegexp("visitor", "[0-9]{10,20}")
	r.PathPrefix("/api/videos/{video_id}").Methods("GET").HandlerFunc(server.GetVideo)
	
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("frontend/static"))))
	r.PathPrefix("/").Methods("GET").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			err := templates.ExecuteTemplate(w, "random.html", nil)
			if err != nil {
				log.Println(err.Error())
			}
		})

	r.Use(loggingMiddleware)

	return server
}

func (s *Server) Start() error {
	return http.Serve(autocert.NewListener(s.domain), s.mux)
}

func (s *Server) CloseDB() error {
	return s.db.Close()
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Do stuff here
        log.Println(r.RequestURI)
        // Call the next handler, which can be another middleware in the chain, or the final handler.
        next.ServeHTTP(w, r)
    })
}
