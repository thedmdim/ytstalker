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
)

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
	mux := http.NewServeMux()
	
	server := &Server{
		Server: http.Server{
			Addr: config.Addr,
			Handler: mux,
		},
		db: db,
		ytr: ytr,
	}

	mux.HandleFunc("/api/random", server.RandomHandler)
	mux.Handle("/static/", LogRequestedUrl(http.StripPrefix("/static/", http.FileServer(http.Dir("frontend/static")))))
	mux.HandleFunc("/", server.PagesHandler)

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
