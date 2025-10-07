package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"zombiezen.com/go/sqlite"
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

	// bind params
	params := r.URL.Query()
	visitor := params.Get("visitor")
	if visitor == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	conn := h.db.Get(context.Background())
	defer h.db.Put(conn)

	var query string
	var stmt *sqlite.Stmt

	query = `
		SELECT videos.id
		FROM videos
		LEFT JOIN videos_visitors vv 
		ON videos.id = vv.video_id AND vv.visitor_id = ? 
	` + ParseQueryParams(params).MakeWhere() + ` 
		ORDER BY 
		CASE WHEN vv.visitor_id IS NULL THEN 0 ELSE vv.number END,
		RANDOM()
		LIMIT 1
	`
	
	stmt = conn.Prep(query)
	stmt.BindText(1, visitor)

	row, err := stmt.Step()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !row {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	videoID := stmt.GetText("id")
	stmt.ClearBindings(); stmt.Reset()

	_, err = w.Write([]byte(videoID))
	if err != nil {
		log.Println(err)
		return
	}
}

