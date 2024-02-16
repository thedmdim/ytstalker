package utils

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
)

var ErrNoClientLeft = fmt.Errorf("no clients left")

type Stream struct {
	io.Reader
	headers http.Header
	clients *StreamClients
}

type StreamClients struct {
	sync.Mutex
	c map[http.ResponseWriter]bool
}

func (sc *StreamClients) Add(client http.ResponseWriter) {
	sc.Lock()
	sc.c[client] = true
	sc.Unlock()
}

func (sc *StreamClients) Remove(client http.ResponseWriter) {
	sc.Lock()
	delete(sc.c, client)
	sc.Unlock()
}

// broadcast stream to clients
func (sc *StreamClients) Write(p []byte) (int, error) {
	if len(sc.c) == 0 {
		return 0, ErrNoClientLeft
	}
	for client := range sc.c {
		log.Printf("%s\n", p)
		client.Header().Set("Content-Type", "image/jpeg")
		client.Header().Set("Content-Length", fmt.Sprint(len(p)))
		_, err := client.Write(p)
		if err != nil {
			sc.Remove(client)
			log.Println("StreamClients.Write:", err)
		}
	}
	return len(p), nil
}

func NewStreamManager() *StreamManager {
	sm := &StreamManager{}
	sm.streams = make(map[string]*Stream)
	return sm
}

type StreamManager struct {
	sync.Mutex
	streams map[string]*Stream
}

func (sm *StreamManager) Stream(url string, client http.ResponseWriter) error {
	stream := sm.streams[url]
	if stream != nil {
		log.Println("there is a stream alredy:", url)
		log.Println("add client")

		// // write headers
		// for name, values := range stream.headers {
		// 	for _, value := range values {
		// 		client.Header().Add(name, value)
		// 	}
		// }
		client.WriteHeader(http.StatusOK)
		stream.clients.Add(client)
		return nil
	}
	log.Println("there were no such stream yet:", url)

	resp, err := http.Get(url)
	if err != nil {
		client.WriteHeader(http.StatusInternalServerError)
		return err
	}


	log.Println("successfully requested a stream")
	stream = &Stream{
		Reader: resp.Body,
		headers: resp.Header,
		clients: &StreamClients{
			c: map[http.ResponseWriter]bool{client: true},
		},
	}

	log.Println("resp headers:", stream.headers)
	log.Println("w headers before:", client.Header())

	// write headers
	for name, values := range stream.headers {
		for _, value := range values {
			client.Header().Add(name, value)
		}
	}

	log.Println("w headers after:", client.Header())


	sm.Lock()
	sm.streams[url] = stream
	sm.Unlock()

	log.Println("start copiing")
	_, err = io.Copy(stream.clients, stream)
	return err
}