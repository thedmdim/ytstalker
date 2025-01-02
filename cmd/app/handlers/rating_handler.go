package handlers

import (
	"fmt"
	"log"
	"net/http"

	"zombiezen.com/go/sqlite"
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

func (s *Router) GetStats(w http.ResponseWriter, r *http.Request) {

	conn := s.db.Get(r.Context())
	defer s.db.Put(conn)

	var err error
	stats := &Stats{}
	stats.Best, err = GetTopRated(conn, true, 10)
	if err != nil {
		log.Println("GetTopRated:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	stats.Worst, err = GetTopRated(conn, false, 10)
	if err != nil {
		log.Println("GetTopRated:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	total, err := TotalVideosNum(conn)
	if err != nil {
		log.Println("GetTopRated:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	stats.Total = total

	Templates.ExecuteTemplate(w, "stats.html", stats)

}

func TotalVideosNum(conn *sqlite.Conn) (int64, error) {
	stmt := conn.Prep(`SELECT COUNT(videos.id) total FROM videos`)

	if _, err := stmt.Step(); err != nil {
		return 0, fmt.Errorf("stmt.ClearBindings: %w", err)
	}
	return stmt.GetInt64("total"), stmt.Reset()
}

func GetTopRated(conn *sqlite.Conn, coolRated bool, limit int64) ([]*RatedVideo, error) {

	var cool int64
	if coolRated {
		cool = 1
	}

	stmt := conn.Prep(`
		SELECT SUM(reactions.cool = ?) AS reactions_sum, reactions.video_id, videos.title
		FROM reactions
		JOIN videos ON videos.id = reactions.video_id
		GROUP BY reactions.video_id
		ORDER BY reactions_sum DESC
		LIMIT ?
	`)

	stmt.BindInt64(1, cool)
	stmt.BindInt64(2, limit)

	var result []*RatedVideo
	for {
		row, err := stmt.Step()
		if err != nil {
			return nil, err
		}

		if row {
			ratedVideo := &RatedVideo{
				ID:        stmt.GetText("video_id"),
				Title:     stmt.GetText("title"),
				Reactions: stmt.GetInt64("reactions_sum"),
			}
			result = append(result, ratedVideo)

		} else {
			break
		}
	}

	if err := stmt.Reset(); err != nil {
		return nil, fmt.Errorf("stmt.Reset: %w", err)
	}
	if err := stmt.ClearBindings(); err != nil {
		return nil, fmt.Errorf("stmt.ClearBindings: %w", err)
	}

	return result, nil
}
