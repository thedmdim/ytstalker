package handlers

import (
	"database/sql"
	"log"
	"net/http"
)

type Stats struct {
	Total int64
	Best  []*RatedVideo
	Worst []*RatedVideo
}

type RatedVideo struct {
	ID        string
	Title     string
	Reactions int64
}

func (s *Handlers) GetStats(w http.ResponseWriter, r *http.Request) {

	var err error
	stats := &Stats{}
	stats.Best, err = GetTopRated(s.db, true, 10)
	if err != nil {
		log.Println("GetTopRated:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	stats.Worst, err = GetTopRated(s.db, false, 10)
	if err != nil {
		log.Println("GetTopRated:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	stats.Total, err = TotalVideosNum(s.db)
	if err != nil {
		log.Println("TotalVideosNum:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s.templates.ExecuteTemplate(w, "stats.html", stats)
}

func TotalVideosNum(db *sql.DB) (int64, error) {
	var total int64
	err := db.QueryRow(`SELECT COUNT(*) FROM videos`).Scan(&total)
	return total, err
}

func GetTopRated(db *sql.DB, coolRated bool, limit int64) ([]*RatedVideo, error) {

	var cool int64
	if coolRated {
		cool = 1
	}

	rows, err := db.Query(`
		SELECT SUM(reactions.cool = ?), reactions.video_id, videos.title
		FROM reactions
		JOIN videos ON videos.id = reactions.video_id
		GROUP BY reactions.video_id
		ORDER BY 1 DESC
		LIMIT ?
	`, cool, limit)
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
