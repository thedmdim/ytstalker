package handlers

import (
	"context"
	"log"
	"net/http"

	"ytstalker/cmd/app/db"
)

type Stats struct {
	Total int
	Best  []*RatedVideo
	Worst []*RatedVideo
}

type RatedVideo struct {
	ID        string
	Title     string
	Reactions int
}

func (s *Handlers) GetStats(w http.ResponseWriter, r *http.Request) {

	var err error
	stats := &Stats{}
	stats.Best, err = GetTopRated(r.Context(), s.db, true, 10)
	if err != nil {
		log.Println("GetTopRated:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	stats.Worst, err = GetTopRated(r.Context(), s.db, false, 10)
	if err != nil {
		log.Println("GetTopRated:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	stmt, err := s.db.Prep("SELECT COUNT(*) FROM videos")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = stmt.QueryRowContext(r.Context()).Scan(&stats.Total)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s.templates.ExecuteTemplate(w, "stats.html", stats)
}

func GetTopRated(ctx context.Context, db *db.DBWrapper, coolRated bool, limit int64) ([]*RatedVideo, error) {

	var cool int64
	if coolRated {
		cool = 1
	}

	stmt, err := db.Prep(`
		SELECT SUM(reactions.cool = ?), reactions.video_id, videos.title
		FROM reactions
		JOIN videos ON videos.id = reactions.video_id
		GROUP BY reactions.video_id
		ORDER BY 1 DESC
		LIMIT ?
	`)

	rows, err := stmt.QueryContext(ctx, cool, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*RatedVideo
	for rows.Next() {
		rv := &RatedVideo{}
		if err := rows.Scan(&rv.Reactions, &rv.ID, &rv.Title); err != nil {
			return nil, err
		}
		result = append(result, rv)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
