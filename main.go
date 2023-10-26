package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"go-youtube-stalker-site/backend/api"
)

func main() {

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	
	server := api.NewServer()

	go func() {
		defer server.Shutdown(ctx)
		log.Println(server.ListenAndServe())
	}()

	<- stop
	
	log.Println("shutting down the server...")
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		log.Fatal("couldn't shut down:", err.Error())
	}
	log.Println("success! thanks for not killing :)")
}
