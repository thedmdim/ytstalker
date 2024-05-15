package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type Message struct {
	Detail string `json:"detail"`
}

type VideoWithReactions struct {
	Video     *Video    `json:"video,omitempty"`
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
	Category  string
	Horizonly bool
	Musiconly bool
}

var ErrNoVideoFound = errors.New("no video found")

func ParseQueryParams(params url.Values) *SearchCriteria {

	sc := &SearchCriteria{}

	viewsValues := strings.Split(params.Get("views"), "-")
	if len(viewsValues) == 2 {
		sc.ViewsFrom = viewsValues[0]
		sc.ViewsTo = viewsValues[1]
	}

	yearsValues := strings.Split(params.Get("years"), "-")
	if len(viewsValues) == 2 {
		sc.YearsFrom = yearsValues[0]
		sc.YearsTo = yearsValues[1]
	}

	category := params.Get("category")
	_, err := strconv.Atoi(category)
	if err == nil {
		sc.Category = category
	}

	horizonly, err := strconv.ParseBool(params.Get("horizonly"))
	if err == nil {
		sc.Horizonly = horizonly
	}

	return sc
}

func (sc *SearchCriteria) MakeWhere() string {
	var conditions []string

	if _, err := strconv.Atoi(sc.ViewsFrom); err == nil {
		conditions = append(conditions, "views >= "+sc.ViewsFrom)
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
	if sc.Horizonly {
		conditions = append(conditions, "vertical = 0")
	}
	if sc.Category != "" {
		conditions = append(conditions, "category = "+sc.Category)
	}
	if len(conditions) > 0 {
		return "AND " + strings.Join(conditions, " AND ")
	}
	return ""
}

func (sc *SearchCriteria) CheckVideo(video *Video) bool {
	if viewsFrom, err := strconv.ParseInt(sc.ViewsFrom, 10, 64); err == nil && video.Views < viewsFrom {
		return false
	}
	if viewsTo, err := strconv.ParseInt(sc.ViewsTo, 10, 64); err == nil && video.Views > viewsTo {
		return false
	}
	if yearFrom, err := strconv.Atoi(sc.YearsFrom); err == nil {
		if video.UploadedAt < time.Date(yearFrom, time.January, 1, 0, 0, 0, 0, time.UTC).Unix() {
			return false
		}
	}
	if yearTo, err := strconv.Atoi(sc.YearsTo); err == nil {
		if video.UploadedAt > time.Date(yearTo, time.December, 31, 0, 0, 0, 0, time.UTC).Unix() {
			return false
		}
	}
	if sc.Horizonly && video.Vertical {
		return false
	}
	if category, err := strconv.ParseInt(sc.Category, 10, 64); err == nil && category != video.Category {
		return false
	}
	return true
}

func TakeFirstUnseen(conn *sqlite.Conn, visitor string, sc *SearchCriteria) (*Video, error) {

	video := &Video{}

	var where string
	if sc != nil {
		where = sc.MakeWhere()
	}

	visitor = strings.ReplaceAll(visitor, " ", "")

	stmt, err := conn.Prepare(fmt.Sprintf(`
		SELECT id, uploaded, title, views, vertical, category
		FROM videos
		WHERE id NOT IN (
			SELECT videos_visitors.video_id
			FROM videos_visitors
			WHERE videos_visitors.visitor_id = %s
		) %s
		ORDER BY random()
		LIMIT 1`,
		visitor,
		where,
	))
	if err != nil {
		return nil, fmt.Errorf("error preparing query: %w", err)
	}
	rows, err := stmt.Step()
	if err != nil {
		return nil, err
	}
	if !rows {
		return nil, ErrNoVideoFound
	}

	video.ID = stmt.GetText("id")
	video.UploadedAt = stmt.GetInt64("uploaded")
	video.Title = stmt.GetText("title")
	video.Views = stmt.GetInt64("views")
	video.Vertical = stmt.GetBool("vertical")
	video.Category = stmt.GetInt64("category")

	err = stmt.Reset()
	if err != nil {
		return video, err
	}

	return video, nil
}

func RememberSeen(conn *sqlite.Conn, visitorId string, videoId string) error {

	endFn, err := sqlitex.ImmediateTransaction(conn)
	if err != nil {
		return fmt.Errorf("error creating a transaction: %w", err)
	}
	defer endFn(&err)

	stmt := conn.Prep(`
		INSERT INTO visitors (id, last_seen)
		VALUES(?, unixepoch())
		ON CONFLICT (id)
		DO UPDATE SET last_seen=unixepoch()
	`)
	stmt.BindText(1, visitorId)
	if _, err = stmt.Step(); err != nil {
		return err
	}
	stmt.ClearBindings(); stmt.Reset()

	stmt = conn.Prep(`INSERT INTO videos_visitors (visitor_id, video_id) VALUES (?, ?);`)
	if err != nil {
		return fmt.Errorf("error preparing query: %w", err)
	}
	stmt.BindText(1, visitorId)
	stmt.BindText(2, videoId)
	if _, err = stmt.Step(); err != nil {
		return err
	}
	stmt.ClearBindings(); stmt.Reset()

	return nil
}

func (s *Router) GetRandom(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)

	visitor := r.Header.Get("visitor")
	params := r.URL.Query()

	conn := s.db.Get(r.Context())
	defer s.db.Put(conn)

	searchCriteria := ParseQueryParams(params)
	video, err := TakeFirstUnseen(conn, visitor, searchCriteria)
	if err != nil {
		log.Println("take first unseen failed:", err.Error())
	}

	response := &VideoWithReactions{}

	if video != nil {
		reactions, err := GetReaction(conn, video.ID)
		if err != nil {
			log.Println("couldn't save reaction:", err.Error())
		}

		response.Video = video
		response.Reactions = reactions

		encoder.Encode(response)
		err = RememberSeen(conn, visitor, video.ID)
		if err != nil {
			log.Println("error remembering seen:", err.Error())
		}
		return
	}
	log.Println("video in db not found")
	w.WriteHeader(http.StatusNotFound)
	encoder.Encode(Message{"no more such videos yet"})
}
