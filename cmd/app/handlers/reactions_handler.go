package handlers

import (
	"database/sql"
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

	_, err := s.db.ExecContext(r.Context(), `
		INSERT INTO reactions (cool, visitor_id, video_id)
		VALUES(?, ?, ?)
		ON CONFLICT (visitor_id, video_id)
		DO UPDATE SET cool=?
	`, reactionBool, visitor, videoID, reactionBool)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	stats, err := GetReaction(s.db, videoID)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	json.NewEncoder(w).Encode(stats)
}

func GetReaction(db *sql.DB, videoID string) (Reactions, error) {
	r := Reactions{}
	err := db.QueryRow(`
		SELECT SUM(cool = 1), SUM(cool = 0) FROM reactions WHERE video_id = ?
	`, videoID).Scan(&r.Cools, &r.Trashes)
	return r, err
}
