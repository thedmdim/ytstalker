package main

import (
	"bufio"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"camstalker/cmd/webapp/handlers"

	"zombiezen.com/go/sqlite/sqlitex"
)

func main() {
	// prepare db
	db, err := sqlitex.NewPool(os.Getenv("DSN"), sqlitex.PoolOptions{PoolSize: 100})
	if err != nil {
		log.Fatal("cannot open db", err)
	}

	conn := db.Get(context.Background())
	if err := sqlitex.ExecuteScript(conn, CreateTablesIfNotExists, nil); err != nil {
		log.Fatal("cannot create db: ", err)
	}
	db.Put(conn)
	log.Println("database ready")

	// read country codes
	file, err := os.Open("countries.csv")
    if err != nil {
        log.Fatal("no countries.csv file")
    }
    defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		columns := strings.Split(line, ",")
		countryName := strings.ToLower(columns[0])
		countryCode := strings.ToLower(columns[1])
		handlers.CountryCodes[countryCode] = countryName
	}

	// make router
	handler := handlers.NewRouter(db)
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
