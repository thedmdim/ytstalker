package main

import (
	"log"
	"os"
	"os/signal"

	"ytstalker/backend/api"
	"ytstalker/backend/conf"
)

func main() {
	config := conf.ParseConfig("conf.json")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	
	server := api.NewServer(config)
	go func(){
		err := server.Start()
		log.Println(err)
	}()
	<- stop
	err := server.CloseDB()
	if err != nil {
		log.Println(err)
	}
	log.Println("successfully closed db")
}
