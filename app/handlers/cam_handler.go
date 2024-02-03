package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

func (s *Router) GetCamera(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)

	vars := mux.Vars(r)

	conn := s.db.Get(r.Context())
	defer s.db.Put(conn)

	res := &CamWithReactions{}

	stmt := conn.Prep(`
		SELECT id, uploaded, title, views, vertical, category
		FROM videos
		WHERE id = ?`)

	stmt.BindText(1, vars["cam_id"])
	row, err := stmt.Step()
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	if !row {
		w.WriteHeader(http.StatusNotFound)
		encoder.Encode(Message{"couldn't find video"})
		return
	}
	res.Cam = &Cam{
		ID:         stmt.GetText("id"),
		Addr: stmt.GetText("addr"),
		Adminka: stmt.GetText("addr"),
		Stream: stmt.GetText("stream"),
		Manufacturer: stmt.GetText("manufacturer"),
		Country: stmt.GetText("country"),
	}
	err = stmt.Reset()
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	res.Reactions, _ = GetReaction(conn, res.Cam.ID)
	encoder.Encode(res)
}
