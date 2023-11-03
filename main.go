package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"ytstalker/backend/api"
	"ytstalker/backend/conf"
)

func main() {

	// read config
	confPath := os.Getenv("CONF_PATH")
	if confPath == "" {
		confPath = "conf.json"
	}
	config := conf.ParseConfig(confPath)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	
	server := api.NewServer(config)
	
	go server.ListenAndServe()
	log.Println("server started!")
	<- stop
	log.Println("got stop signal")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	log.Println("shutting down the server...")
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		log.Fatal("couldn't shut down: ", err.Error())
	}
	log.Println("success! thanks for not killing :)")
}
