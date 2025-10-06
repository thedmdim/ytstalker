package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ytstalker/cmd/app/handlers"
	"ytstalker/cmd/app/youtube"
	"github.com/gorilla/mux"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func main() {

	// prepare db
	dsn := os.Getenv("DSN")
	if dsn == "" {
		dsn = "server.db"
	}

	db, err := sqlitex.NewPool(dsn, sqlitex.PoolOptions{PoolSize: 100})
	if err != nil {
		log.Fatal("cannot open db", err)
	}

	conn := db.Get(context.Background())
	if err := sqlitex.ExecuteScript(conn, CreateTablesIfNotExists, nil); err != nil {
		log.Fatal("cannot create db: ", err)
	}

	for _, q := range Migrations {
		if err := sqlitex.ExecuteScript(conn, q, nil); err != nil {
			log.Println("migration error: ", err)
		}
	}

	db.Put(conn)
	log.Println("database ready")

	// init youtube api requester
	ytApiKey := os.Getenv("YT_API_KEY")
	if ytApiKey == "" {
		log.Fatal("You forgot to provide YouTube API key!")
	}
	ytr := youtube.NewYouTubeRequester(ytApiKey)

	// make router
	handlers := handlers.NewHandlers(db)
	router := mux.NewRouter()

	// api
	router.PathPrefix("/api/videos/random").Methods("GET").HandlerFunc(handlers.GetRandom)
	router.PathPrefix("/api/videos/{video_id}/reactions/{visitor}/{reaction:(?:cool|trash)}").Methods("POST").HandlerFunc(handlers.WriteReaction)
	router.PathPrefix("/api/videos/{video_id}").Methods("GET").HandlerFunc(handlers.GetVideoData)
	// pages
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", handlers.CacheHeader(http.FileServer(http.Dir("web/static")))))
	router.PathPrefix("/stats").Methods("GET").HandlerFunc(handlers.GetStats)
	router.PathPrefix("/").Methods("GET").HandlerFunc(handlers.GetVideoPage)

	router.Use(handlers.LoggingMiddleware)


	server := &http.Server{
		Handler: router,
	}

	// search random video in background
	if os.Getenv("LOCAL") == "" {
		go func() {
			for {
				results, err := ytr.FindRandomVideos()
				if err != nil {
					log.Println("background random search:", err.Error())
					continue
				}
	
				conn := db.Get(context.Background())
				err = StoreVideos(conn, results)
				if err != nil {
					log.Println("background random search: couldn't store found videos:", err.Error())
				} else {
					counters := make(map[int]int)
					for _, video := range results {
						year := time.Unix(video.UploadedAt, 0).Year()
						counters[year]++
					}
					log.Println("background random search:", len(results), "found videos stored:")
					for year, n := range counters {
						log.Printf("%d videos from %d\n", n, year)
					}
				}
				db.Put(conn)
	
				time.Sleep(time.Hour / 2)
			}
		}()
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

func StoreVideos(conn *sqlite.Conn, videos map[string]*youtube.Video) error {

	endFn, err := sqlitex.ImmediateTransaction(conn)
	if err != nil {
		return fmt.Errorf("error creating a transaction: %w", err)
	}
	defer endFn(&err)

	stmt := conn.Prep("INSERT INTO videos (id, uploaded, title, views, vertical, category) VALUES (?, ?, ?, ?, ?, ?);")
	for _, video := range videos {

		stmt.BindText(1, video.ID)
		stmt.BindInt64(2, video.UploadedAt)
		stmt.BindText(3, video.Title)
		stmt.BindInt64(4, int64(video.Views))
		stmt.BindBool(5, video.Vertical)
		stmt.BindInt64(6, int64(video.Category))

		if _, err := stmt.Step(); err != nil {
			return fmt.Errorf("stmt.Step: %w", err)
		}
		if err := stmt.Reset(); err != nil {
			return fmt.Errorf("stmt.Reset: %w", err)
		}
		if err := stmt.ClearBindings(); err != nil {
			return fmt.Errorf("stmt.ClearBindings: %w", err)
		}
	}
	return nil
}