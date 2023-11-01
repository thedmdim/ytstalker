package api

import (
	"context"
	"fmt"
	"go-youtube-stalker-site/backend/conf"
	"go-youtube-stalker-site/backend/youtube"
	"log"
	"net/http"
	"os"

	"zombiezen.com/go/sqlite/sqlitex"
	"github.com/gorilla/mux"
	"html/template"
)

var templates = template.Must(template.ParseGlob("frontend/*/*.html"))

type Server struct {
	http.Server
	db *sqlitex.Pool
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
			Addr: config.Addr,
			Handler: r,
		},
		db: db,
		ytr: ytr,
	}

	r.PathPrefix("/api/videos/{video_id}/{reaction:(?:cool|trash)})").Methods("POST").HandlerFunc(server.WriteReaction).HeadersRegexp("visitor", "[0-9]{10,20}")
	r.PathPrefix("/api/videos/random").Methods("GET").HandlerFunc(server.RandomHandler).HeadersRegexp("visitor", "[0-9]{10,20}")

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("frontend/static"))))
	r.PathPrefix("/").Methods("GET").HandlerFunc(
		func (w http.ResponseWriter, r *http.Request) {
			err := templates.ExecuteTemplate(w, "random.html", nil)
			if err != nil {
				log.Println(err.Error())
			}
		})

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

func LogRequestedUrl(h http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("new request:", r.URL.Path)
		h.ServeHTTP(w, r)
	})
}
