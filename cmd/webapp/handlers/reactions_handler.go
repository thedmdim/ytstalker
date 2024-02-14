package handlers

import (
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type ReactionStats struct {
	Likes   int64 `json:"likes"`
	Dislikes int64 `json:"dislikes"`
}

func (s *Router) WriteReaction(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)

	vars := mux.Vars(r)
	camID, err := hex.DecodeString(vars["cam_id"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	visitor := r.Header.Get("visitor")
	reaction := vars["reaction"]

	conn := s.db.Get(r.Context())
	defer s.db.Put(conn)

	stmt := conn.Prep(`
		INSERT INTO reactions (like, visitor_id, cam_id)
		VALUES(?, ?, ?)
		ON CONFLICT (visitor_id, cam_id)
		DO UPDATE SET like=?
	`)

	var reactionBool bool
	if reaction == "cool" { reactionBool = true }

	stmt.BindBool(1, reactionBool)
	stmt.BindText(2, visitor)
	stmt.BindBytes(3, camID)
	stmt.BindBool(4, reactionBool)
	stmt.BindBool(5, reactionBool)

	stmt.Step(); stmt.ClearBindings(); stmt.Reset()

	stmt = conn.Prep(`
		SELECT
			SUM(reactions.like = 1) like, SUM(reactions.like = 1) dislike
		FROM reactions
		WHERE reactions.cam_id = ?
	`)

	stmt.BindBytes(1, camID)
	_, err = stmt.Step()
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	response := ReactionStats{
		Likes: stmt.GetInt64("likes"),
		Dislikes: stmt.GetInt64("dislikes"),
	}
	stmt.Reset(); stmt.ClearBindings()

	encoder.Encode(response)
}
