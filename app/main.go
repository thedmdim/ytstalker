package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ytstalker/app/api"
	"ytstalker/app/conf"
	"ytstalker/app/youtube"

	"zombiezen.com/go/sqlite/sqlitex"
)

func main() {
	config := conf.ParseConfig()

	// prepare db
	db, err := sqlitex.Open(config.DSN, 0, config.DbPoolSize)
	if err != nil {
		log.Fatal("cannot open db", err)
	}

	conn := db.Get(context.Background())
	if err := sqlitex.ExecuteScript(conn, api.CreateTablesIfNotExists, nil); err != nil {
		log.Fatal("cannot create db: ", err)
	}
	db.Put(conn)
	log.Println("database ready")

	// init youtube api requester
	ytr := youtube.NewYouTubeRequester(config)

	// make router
	handler := api.NewRouter(db, ytr)
	server := &http.Server{
		Handler: handler,
	}

	// serve 80
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	// gracefull shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	log.Println("successfully finished serving")

	err = db.Close()
	if err != nil {
		log.Fatalln("error gracefully closing db:", err.Error())
	}
	log.Println("successfully closed db", "\n", "thanks :)")

}
