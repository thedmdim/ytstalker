package api

import (
	"context"
	"database/sql"
	"go-youtube-stalker-site/backend/conf"
	"go-youtube-stalker-site/backend/youtube"
	"log"
	"net/http"
	"os"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type Server struct {
	http.Server
	wlock sync.Mutex
	db *sql.DB
	ytr *youtube.YouTubeRequester
}

func NewServer() *Server {

	// read config
	config := conf.ParseConfig("conf.json")

	// init db
	db, err := sql.Open("sqlite3", config.DbName)
	if err != nil {
		log.Fatal("cannot open db", err)
	}
	dbScheme, err := os.ReadFile("db.sql")
	if err != nil {
		log.Fatal("cannot open db.sql: ", err.Error())
	}
	_, err = db.Exec(string(dbScheme))
	if err != nil {
		db.Close()
		log.Fatal("cannot create db: ", err)
	}
	log.Println("database ready")

	// init youtube requester
	ytr := youtube.NewYouTubeRequester(config)

	server := &Server{
		db: db,
		ytr: ytr,
	}

	// Create a new router
	mux := http.NewServeMux()
	mux.HandleFunc("/api/random", server.Random)
	mux.Handle("/static/", LogRequestedUrl(http.StripPrefix("/static/", http.FileServer(http.Dir("frontend/static")))))
	mux.HandleFunc("/", server.ServePages)

	server.Server = http.Server{
		Addr: config.Addr,
		Handler: mux,
	}

	return server
}

func (s *Server) Shutdown(ctx context.Context) error {
	err := s.Server.Shutdown(ctx)
	if err != nil {
		return err
	}
	err = s.db.Close()
	if err != nil {
		return err
	}
	return nil
}

func LogRequestedUrl(h http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("new request:", r.URL.Path)
		h.ServeHTTP(w, r)
	})
}
