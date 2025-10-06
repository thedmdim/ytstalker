package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"zombiezen.com/go/sqlite"
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

	conn := h.db.Get(context.Background())
	defer h.db.Put(conn)

	// bind params
	vars := mux.Vars(r)
	videoID := vars["video_id"]

	var query string
	var stmt *sqlite.Stmt

	query = `
		SELECT
			videos.id,
			videos.uploaded,
			videos.title,
			videos.views,
			videos.vertical,
			videos.category,
			SUM(cool = 1) AS cools,
			SUM(cool = 0) AS trashes
		FROM videos
		LEFT JOIN reactions ON reactions.video_id = videos.id
		WHERE videos.id = ?
	`
	stmt = conn.Prep(query)
	stmt.BindText(1, videoID)
	
	row, err := stmt.Step()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !row {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	response := VideoWithReactions{
		Video: Video{
			ID: stmt.GetText("id"),
			UploadedAt: stmt.GetInt64("uploaded"),
			Title: stmt.GetText("title"),
			Views: stmt.GetInt64("views"),
			Vertical: stmt.GetBool("vertical"),
			Category: stmt.GetInt64("category"),
		},
		Reactions: Reactions{
			Cools: stmt.GetInt64("cools"),
			Trashes: stmt.GetInt64("trashes"),
		},
	}

	stmt.ClearBindings(); stmt.Reset()

	// w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil { log.Println(err) }

	// remember seen
	stmt = conn.Prep(`
		INSERT INTO visitors (id, last_seen)
		VALUES(?, unixepoch())
		ON CONFLICT (id)
		DO UPDATE SET last_seen=unixepoch()
	`)
	stmt.BindText(1, visitor)
	if _, err = stmt.Step(); err != nil {
		log.Println(err)
		return
	}
	stmt.ClearBindings(); stmt.Reset()

	stmt = conn.Prep(`
		INSERT INTO videos_visitors (visitor_id, video_id)
		VALUES (?, ?)
		ON CONFLICT (visitor_id, video_id)
		DO UPDATE SET number = number + 1
	`)
	stmt.BindText(1, visitor)
	stmt.BindText(2, videoID)
	if _, err = stmt.Step(); err != nil {
		log.Println(err)
		return
	}
	stmt.ClearBindings(); stmt.Reset()
}

