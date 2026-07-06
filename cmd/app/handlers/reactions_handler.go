package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type ReactionStats struct {
	Cools   int64 `json:"cools"`
	Trashes int64 `json:"trashes"`
}

func (s *Handlers) WriteReaction(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	videoID := vars["video_id"]
	visitor := vars["visitor"]
	reaction := vars["reaction"]

	var reactionBool bool
	if reaction == "cool" {
		reactionBool = true
	}

	stmt, err := s.db.Prep(`
		INSERT INTO reactions (cool, visitor_id, video_id)
		VALUES(?, ?, ?)
		ON CONFLICT (visitor_id, video_id)
		DO UPDATE SET cool=?
	`)


	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	_, err = stmt.ExecContext(r.Context(), reactionBool, visitor, videoID, reactionBool)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}


	stmt, err = s.db.Prep("SELECT SUM(cool = 1), SUM(cool = 0) FROM reactions WHERE video_id = ?")
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	stats := Reactions{}
	err = stmt.QueryRowContext(r.Context(), videoID).Scan(&stats.Cools, &stats.Trashes)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	
	json.NewEncoder(w).Encode(stats)
}
