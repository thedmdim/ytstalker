package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

func (s *Router) GetVideo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)

	vars := mux.Vars(r)

	conn := s.db.Get(r.Context())
	defer s.db.Put(conn)

	res := &VideoWithReactions{}

	stmt := conn.Prep(`
		SELECT id, uploaded, title, views, vertical, category
		FROM videos
		WHERE id = ?`)

	stmt.BindText(1, vars["video_id"])
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
	res.Video = &Video{
		ID:         stmt.GetText("id"),
		UploadedAt: stmt.GetInt64("uploaded"),
		Title:      stmt.GetText("title"),
		Views:      stmt.GetInt64("views"),
		Vertical:   stmt.GetBool("vertical"),
		Category:   stmt.GetInt64("category"),
	}
	err = stmt.Reset()
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	
	res.Reactions, _ = GetReaction(conn, res.Video.ID)
	encoder.Encode(res)
}
