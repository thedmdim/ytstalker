package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"zombiezen.com/go/sqlite"
)

type ReactionStats struct {
	Cools   int64 `json:"cools"`
	Trashes int64 `json:"trashes"`
}

func (s *Router) WriteReaction(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)

	vars := mux.Vars(r)

	visitor := r.Header.Get("visitor")
	videoID := vars["video_id"]
	reaction := vars["reaction"]

	conn := s.db.Get(r.Context())
	defer s.db.Put(conn)

	stmt := conn.Prep(`
		INSERT INTO reactions (cool, visitor_id, video_id)
		VALUES(?, ?, ?)
		ON CONFLICT (visitor_id, video_id)
		DO UPDATE SET cool=?
	`)

	var reactionBool bool
	if reaction == "cool" {
		reactionBool = true
	}

	stmt.BindBool(1, reactionBool)
	stmt.BindText(2, visitor)
	stmt.BindText(3, videoID)
	stmt.BindBool(4, reactionBool)
	stmt.Step()
	stmt.Reset()

	stats, err := GetReactionStats(conn, videoID)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		encoder.Encode(Message{"couldn't save reaction"})
		return
	}
	encoder.Encode(stats)
}

func GetReactionStats(conn *sqlite.Conn, videoID string) (Reactions, error) {
	r := Reactions{}

	stmt := conn.Prep(`
			SELECT SUM(cool = 1) AS cools, SUM(cool = 0) AS trashes
			FROM reactions
			WHERE video_id = ?
	`)
	stmt.BindText(1, videoID)
	_, err := stmt.Step()
	if err != nil {
		return r, err
	}

	r.Cools = stmt.GetInt64("cools")
	r.Trashes = stmt.GetInt64("trashes")

	err = stmt.Reset()
	if err != nil {
		return r, err
	}

	return r, nil
}
