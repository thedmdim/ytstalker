package utils

import (
	"bytes"
	"errors"
	"regexp"
	"io"
	"net/http"
	"sync"
)

var boundaryRe = regexp.MustCompile(`boundary="([^"]+)"`)
var ErrNoClientLeft = errors.New("no clients left")

type stream struct {
	so io.Reader
	headers http.Header
	boundary string
	mu sync.Mutex
	clients map[http.ResponseWriter]bool
	wg sync.WaitGroup
}

// broadcast stream to clients
func (s *stream) Write(p []byte) (int, error) {
	if len(s.clients) == 0 {
		return 0, ErrNoClientLeft
	}
	for client, trimmed := range s.clients {
		if !trimmed {
			index := bytes.Index(p, []byte("--" + s.boundary))
			if index != -1 {
				p = p[index:]
			}
			s.setTrimmed(client)
		}

		_, err := client.Write(p)
		if err != nil {
			s.del(client)
		}
	}
	return len(p), nil
}

func (s *stream) addClient(client http.ResponseWriter) {
	s.mu.Lock()
	s.clients[client] = false
	s.mu.Unlock()
}

func (s *stream) del(client http.ResponseWriter) {
	s.mu.Lock()
	delete(s.clients, client)
	s.mu.Unlock()
}

func (s *stream) setTrimmed(client http.ResponseWriter) {
	s.mu.Lock()
	s.clients[client] = true
	s.mu.Unlock()
}

func NewStreamManager() *StreamManager {
	sm := &StreamManager{}
	sm.streams = make(map[string]*stream)
	return sm
}

type StreamManager struct {
	sync.Mutex
	streams map[string]*stream
}

func (sm *StreamManager) Stream(url string, client http.ResponseWriter) error {

	var err error

	st, ok := sm.streams[url]
	if !ok {
		resp, err := http.Get(url)
		if err != nil {
			client.WriteHeader(http.StatusInternalServerError)
			return err
		}

		st = &stream{
			so: resp.Body,
			headers: resp.Header,
			clients: make(map[http.ResponseWriter]bool),
		}

		match := boundaryRe.FindStringSubmatch(resp.Header.Get("content-type"))
		if len(match) > 1 {
			st.boundary = match[1]
		}

		sm.Lock()
		sm.streams[url] = st
		sm.Unlock()
	}

	// write headers
	for name, values := range st.headers {
		for _, value := range values {
			client.Header().Set(name, value)
		}
	}

	st.addClient(client)

	if !ok {
		st.wg.Add(1)
		_, err = io.Copy(st, st.so)
		st.wg.Done()
	}

	// wait until stream wokring
	st.wg.Wait()

	return err
}