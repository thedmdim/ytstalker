package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Message struct {
	Detail string `json:"detail"`
}

type VideoWithReactions struct {
	Video     Video    `json:"video"`
	Reactions Reactions `json:"reactions"`
}

type Video struct {
	ID         string `json:"id"`
	UploadedAt int64  `json:"uploaded"`
	Title      string `json:"title"`
	Views      int64  `json:"views"`
	Vertical   bool   `json:"vertical"`
	Category   int64  `json:"category"`
}

type Reactions struct {
	Cools   int64 `json:"cools"`
	Trashes int64 `json:"trashes"`
}

type SearchCriteria struct {
	ViewsFrom string
	ViewsTo   string
	YearsFrom string
	YearsTo   string
	Year      string
	Category  string
	Horizonly bool
	Musiconly bool
}

var ErrNoVideoFound = errors.New("no video found")

func ParseQueryParams(params url.Values) SearchCriteria {

	sc := SearchCriteria{}

	viewsValues := strings.Split(params.Get("views"), "-")
	if len(viewsValues) == 2 {
		sc.ViewsFrom = viewsValues[0]
		sc.ViewsTo = viewsValues[1]
	}

	yearsValues := strings.Split(params.Get("years"), "-")
	if len(yearsValues) == 2 {
		sc.YearsFrom = yearsValues[0]
		sc.YearsTo = yearsValues[1]
	}

	year := params.Get("year")
	_, err := strconv.Atoi(year)
	if err == nil {
		sc.Year = year
	}

	category := params.Get("category")
	_, err = strconv.Atoi(category)
	if err == nil {
		sc.Category = category
	}

	horizonly, err := strconv.ParseBool(params.Get("horizonly"))
	if err == nil {
		sc.Horizonly = horizonly
	}

	return sc
}

func (sc SearchCriteria) MakeWhere() string {
	var conditions []string

	if _, err := strconv.Atoi(sc.ViewsFrom); err == nil {
		conditions = append(conditions, "videos.views >= "+sc.ViewsFrom)
	}
	if _, err := strconv.Atoi(sc.ViewsTo); err == nil {
		conditions = append(conditions, "views <= "+sc.ViewsTo)
	}
	if year, err := strconv.Atoi(sc.YearsFrom); err == nil {
		timestamp := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
		conditions = append(conditions, fmt.Sprintf("uploaded >= %d", timestamp))
	}
	if year, err := strconv.Atoi(sc.YearsTo); err == nil {
		timestamp := time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC).Unix()
		conditions = append(conditions, fmt.Sprintf("uploaded <= %d", timestamp))
	}
	if year, err := strconv.Atoi(sc.Year); err == nil {
		start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
		end := time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
		conditions = append(conditions, fmt.Sprintf("uploaded >= %d AND uploaded < %d", start, end))
	}

	if sc.Horizonly {
		conditions = append(conditions, "videos.vertical = 0")
	}
	if sc.Category != "" {
		conditions = append(conditions, "videos.category = "+sc.Category)
	}
	if len(conditions) > 0 {
		return "WHERE " + strings.Join(conditions, " AND ")
	}
	return ""
}

func (h Handlers) GetRandom(w http.ResponseWriter, r *http.Request) {

	r.Body.Close()

	params := r.URL.Query()
	visitor := params.Get("visitor")
	if visitor == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	query := `
		WITH filtered AS (
			SELECT videos.id, vv.number, vv.visitor_id
			FROM videos
			LEFT JOIN videos_visitors vv 
			ON videos.id = vv.video_id AND vv.visitor_id = ?
			` + ParseQueryParams(params).MakeWhere() + `
		),

		ceil AS (SELECT COUNT(*) n FROM filtered)

		SELECT id 
		FROM filtered
		ORDER BY CASE WHEN visitor_id IS NULL THEN 0 ELSE number END
		LIMIT 1 
		OFFSET (SELECT ABS(RANDOM() % ceil.n) FROM ceil);
	`

	var videoID string
	stmt, err := h.db.Prep(query)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = stmt.QueryRowContext(r.Context(), visitor).Scan(&videoID)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write([]byte(videoID))
	if err != nil {
		log.Println(err)
	}
}
