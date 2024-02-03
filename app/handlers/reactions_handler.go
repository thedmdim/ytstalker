package handlers

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
	camID := vars["cam_id"]
	reaction := vars["reaction"]

	conn := s.db.Get(r.Context())
	defer s.db.Put(conn)

	stmt := conn.Prep(`
		INSERT INTO reactions (cool, visitor_id, cam_id)
		VALUES(?, ?, ?)
		ON CONFLICT (visitor_id, cam_id)
		DO UPDATE SET cool=?
	`)

	var reactionBool bool
	if reaction == "cool" {
		reactionBool = true
	}

	stmt.BindBool(1, reactionBool)
	stmt.BindText(2, visitor)
	stmt.BindText(3, camID)
	stmt.BindBool(4, reactionBool)
	stmt.Step()
	stmt.Reset()

	stats, err := GetReaction(conn, camID)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		encoder.Encode(Message{"couldn't save reaction"})
		return
	}
	encoder.Encode(stats)
}

func GetReaction(conn *sqlite.Conn, camID string) (Reactions, error) {
	r := Reactions{}

	stmt := conn.Prep(`
			SELECT SUM(cool = 1) AS cools, SUM(cool = 0) AS trashes
			FROM reactions
			WHERE cam_id = ?
	`)
	stmt.BindText(1, camID)
	_, err := stmt.Step()
	if err != nil {
		return r, err
	}

	r.Cools = stmt.GetInt64("cools")
	r.Trashes = stmt.GetInt64("trashes")

	stmt.Reset()
	stmt.ClearBindings()

	return r, nil
}
