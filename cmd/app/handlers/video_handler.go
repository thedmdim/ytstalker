package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func (s Handlers) GetVideoPage(w http.ResponseWriter, r *http.Request) {
	err := s.templates.ExecuteTemplate(w, "video.html", nil)
	if err != nil {
		log.Println(err.Error())
	}
}

func (h Handlers) GetVideoData(w http.ResponseWriter, r *http.Request) {

	r.Body.Close()

	params := r.URL.Query()
	visitor := params.Get("visitor")
	if visitor == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	videoID := vars["video_id"]


	stmt, err := h.db.Prep(`
		SELECT videos.id, videos.uploaded, videos.title, videos.views,
		       videos.vertical, videos.category,
		       COALESCE(SUM(cool = 1), 0),
		       COALESCE(SUM(cool = 0), 0)
		FROM videos
		LEFT JOIN reactions ON reactions.video_id = videos.id
		WHERE videos.id = ?
	`)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var video VideoWithReactions
	err = stmt.QueryRowContext(r.Context(), videoID).Scan(
		&video.Video.ID,
		&video.Video.UploadedAt,
		&video.Video.Title,
		&video.Video.Views,
		&video.Video.Vertical,
		&video.Video.Category,
		&video.Reactions.Cools,
		&video.Reactions.Trashes,
	)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(video)
	if err != nil {
		log.Println(err)
	}

	stmt, err = h.db.Prep(`
		INSERT INTO visitors (id, last_seen)
		VALUES(?, unixepoch())
		ON CONFLICT (id)
		DO UPDATE SET last_seen=unixepoch()
	`)
	if err != nil {
		log.Println(err)
		return
	}

	_, err = stmt.Exec(visitor)
	if err != nil {
		log.Println(err)
		return
	}

	stmt, err = h.db.Prep(`
		INSERT INTO videos_visitors (visitor_id, video_id)
		VALUES (?, ?)
		ON CONFLICT (visitor_id, video_id)
		DO UPDATE SET number = number + 1
	`)
	_, err = stmt.Exec(visitor, videoID)
	if err != nil {
		log.Println(err)
		return
	}
}
