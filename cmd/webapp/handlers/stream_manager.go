package handlers

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
		_, err := client.Write(p)
		if err != nil {
			sc.Remove(client)
			log.Println("StreamClients.Write:", err)
		}
	}
	return len(p), nil
}

type StreamManager struct {
	sync.Mutex
	streams map[string]*Stream
}

func (sm *StreamManager) Stream(url string, client http.ResponseWriter) error {
	stream := sm.streams[url]
	if stream != nil {
		// write headers
		for name, values := range stream.headers {
			for _, value := range values {
				client.Header().Add(name, value)
			}
		}
		stream.clients.Add(client)
		return nil
	}

	resp, err := http.Get(url)
	if err != nil {
		client.WriteHeader(http.StatusInternalServerError)
		return err
	}
	stream = &Stream{
		Reader: resp.Body,
		headers: resp.Header,
		clients: &StreamClients{
			c: map[http.ResponseWriter]bool{client: true},
		},
	}

	sm.Lock()
	sm.streams[url] = stream
	sm.Lock()

	_, err = io.Copy(stream.clients, stream)
	return err
}