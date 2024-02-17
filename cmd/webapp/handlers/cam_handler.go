package handlers

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"camstalker/cmd/webapp/utils"

	"github.com/gorilla/mux"
)

var streamManager = utils.NewStreamManager()

type Cam struct {
	ID           string
	Addr         string
	Adminka      string
	Stream       string
	Model        string
	Country      string
	Likes        int64
	Dislikes     int64
}

func (s *Router) GetCam(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	camID, err := hex.DecodeString(vars["cam_id"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	conn := s.db.Get(context.Background())
	defer s.db.Put(conn)

	stmt := conn.Prep(`
		SELECT
			cams.addr,
			cams.adminka,
			cams.stream,
			cams.model, 
			cams.country,
			SUM(reactions.like = 1) AS likes,
			SUM(reactions.like = 0) AS dislikes
		FROM cams
		LEFT JOIN reactions ON reactions.cam_id = cams.id
		WHERE cams.id = ?`)
		
	stmt.BindBytes(1, camID)

	row, err := stmt.Step()
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	if !row {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	cam := &Cam{
		ID:           vars["cam_id"],
		Addr:         stmt.GetText("addr"),
		Adminka:      stmt.GetText("adminka"),
		Stream:       stmt.GetText("stream"),
		Model:        stmt.GetText("model"),
		Country:      stmt.GetText("country"),
		Likes:        stmt.GetInt64("likes"),
		Dislikes:     stmt.GetInt64("dislikes"),
	}
	stmt.ClearBindings(); stmt.Reset()

	templates.ExecuteTemplate(w, "cam.html", cam)

	// after we sent page, add cam to seen ones
	visitor := r.Header.Get("visitor")
	stmt = conn.Prep(`
		INSERT INTO visitors (id, last_seen)
		VALUES(?, unixepoch())
		ON CONFLICT (id)
		DO UPDATE SET last_seen=unixepoch()
	`)
	stmt.BindText(1, visitor)
	stmt.Step(); stmt.ClearBindings(); stmt.Reset()

	stmt = conn.Prep(`INSERT INTO cams_visitors (visitor_id, cam_id) VALUES (?, ?);`)
	stmt.BindText(1, visitor)
	stmt.BindBytes(2, camID)
	stmt.Step(); stmt.ClearBindings(); stmt.Reset()
}


func (s *Router) RedirectRandom(w http.ResponseWriter, r *http.Request) {

	conn := s.db.Get(r.Context())
	defer s.db.Put(conn)

	// build query
	query := `
		SELECT id
		FROM cams
		WHERE id NOT IN (
			SELECT cam_id
			FROM cams_visitors
			WHERE visitor_id = ?
		)`

	params := r.URL.Query()
	visitor := r.Header.Get("visitor")

	model := params.Get("model")
	if model != "" { query += " AND model = ?" }

	country := params.Get("country")
	if country != "" { query += " AND country = ?" }

	query += " ORDER BY random() LIMIT 1"

	// bind params
	stmt := conn.Prep(query)
	stmt.BindText(1, visitor)
	
	if model != "" && country != "" { 
		stmt.BindText(2, model)
		stmt.BindText(3, country)
	} else if model != "" {
		stmt.BindText(2, model)
	} else if country != "" {
		stmt.BindText(2, country)
	}

	row, err := stmt.Step()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// TO DO : if not row, select seen
	if !row {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var camID [16]byte
	stmt.GetBytes("id", camID[:])

	stmt.Reset(); stmt.ClearBindings()
	
    http.Redirect(w, r, fmt.Sprintf("/%x", camID), http.StatusMovedPermanently)
}

func (s *Router) ProxyStream(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	camID, err := hex.DecodeString(vars["cam_id"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	conn := s.db.Get(r.Context())

	stmt := conn.Prep("SELECT addr, stream FROM cams WHERE id = ?")
	stmt.BindBytes(1, camID)

	row, err := stmt.Step()
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	if !row {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	streamURL := "http://" + stmt.GetText("addr") + stmt.GetText("stream")

	stmt.ClearBindings(); stmt.Reset()
	s.db.Put(conn)
	
	streamManager.Stream(streamURL, w)
}
