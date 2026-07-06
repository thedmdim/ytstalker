package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ytstalker/cmd/app/handlers"
	"ytstalker/cmd/app/db"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

func main() {

	dsn := os.Getenv("DSN")
	if dsn == "" {
		dsn = "server.db"
	}

	dbw := db.NewDBWrapper(dsn)

	if _, err := dbw.Exec(db.CreateTablesIfNotExists); err != nil {
		log.Fatal("cannot create db: ", err)
	}

	for _, q := range db.Migrations {
		if _, err := dbw.Exec(q); err != nil {
			log.Println("migration error: ", err)
		}
	}
	log.Println("database ready")

	handlers := handlers.NewHandlers(dbw)
	router := mux.NewRouter()

	router.PathPrefix("/api/videos/random").Methods("GET").HandlerFunc(handlers.GetRandom)
	router.PathPrefix("/api/videos/{video_id}/reactions/{visitor}/{reaction:(?:cool|trash)}").Methods("POST").HandlerFunc(handlers.WriteReaction)
	router.PathPrefix("/api/videos/{video_id}").Methods("GET").HandlerFunc(handlers.GetVideoData)
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", handlers.CacheHeader(http.FileServer(http.Dir("web/static")))))
	router.PathPrefix("/stats").Methods("GET").HandlerFunc(handlers.GetStats)
	router.PathPrefix("/").Methods("GET").HandlerFunc(handlers.GetVideoPage)

	router.Use(handlers.LoggingMiddleware)

	server := &http.Server{
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	log.Println("successfully finished serving")

	err := dbw.Close()
	if err != nil {
		log.Fatalln("error gracefully closing db:", err.Error())
	}
	log.Println("successfully closed db", "\n", "thanks :)")
}