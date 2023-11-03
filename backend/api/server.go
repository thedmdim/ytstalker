package api

import (
	"context"
	"fmt"
	"ytstalker/backend/conf"
	"ytstalker/backend/youtube"
	"log"
	"net/http"
	"os"

	"html/template"

	"github.com/gorilla/mux"
	"zombiezen.com/go/sqlite/sqlitex"
)

var templates = template.Must(template.ParseGlob("frontend/*/*.html"))

type Server struct {
	http.Server
	db  *sqlitex.Pool
	ytr *youtube.YouTubeRequester
}

func NewServer() *Server {

	// read config
	config := conf.ParseConfig("conf.json")

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
		Server: http.Server{
			Addr:    config.Addr,
			Handler: r,
		},
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

func (s *Server) Shutdown(ctx context.Context) error {
	err := s.Server.Shutdown(ctx)
	if err != nil {
		return err
	}
	err = s.db.Close()
	if err != nil {
		return fmt.Errorf("error closing db conn pool: %w", err)
	}
	return nil
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Do stuff here
        log.Println(r.RequestURI)
        // Call the next handler, which can be another middleware in the chain, or the final handler.
        next.ServeHTTP(w, r)
    })
}
