package handlers

import (
	"context"
	"encoding/json"
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
		timestamp := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
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
		if video.UploadedAt > time.Date(yearTo, time.January, 1, 0, 0, 0, 0, time.UTC).Unix() {
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

func (s *Router) TakeFirstUnseen(conn *sqlite.Conn, sc *SearchCriteria, visitor string) (*Video, error) {

	video := &Video{}

	stmt, _, err := conn.PrepareTransient(fmt.Sprintf(`
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
		sc.MakeWhere(),
	))
	if err != nil {
		return nil, fmt.Errorf("error preparing query: %w", err)
	}
	rows, err := stmt.Step()
	if err != nil {
		return nil, err
	}
	if !rows {
		return nil, nil
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

// func (s *Router) StoreVideos(conn *sqlite.Conn, videos map[string]*Video) error {

// 	endFn, err := sqlitex.ImmediateTransaction(conn)
// 	if err != nil {
// 		return fmt.Errorf("error creating a transaction: %w", err)
// 	}
// 	defer endFn(&err)

// 	stmt := conn.Prep("INSERT INTO videos (id, uploaded, title, views, vertical, category) VALUES (?, ?, ?, ?, ?, ?);")
// 	for _, video := range videos {

// 		stmt.BindText(1, video.ID)
// 		stmt.BindInt64(2, video.UploadedAt)
// 		stmt.BindText(3, video.Title)
// 		stmt.BindInt64(4, int64(video.Views))
// 		stmt.BindBool(5, video.Vertical)
// 		stmt.BindInt64(6, int64(video.Category))

// 		if _, err := stmt.Step(); err != nil {
// 			return fmt.Errorf("stmt.Step: %w", err)
// 		}
// 		if err := stmt.Reset(); err != nil {
// 			return fmt.Errorf("stmt.Reset: %w", err)
// 		}
// 		if err := stmt.ClearBindings(); err != nil {
// 			return fmt.Errorf("stmt.ClearBindings: %w", err)
// 		}
// 	}
// 	return nil
// }

func (s *Router) RememberSeen(conn *sqlite.Conn, visitorId string, videoId string) error {

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

	stmt, err = conn.Prepare(`INSERT INTO videos_visitors (visitor_id, video_id) VALUES (?, ?);`)
	if err != nil {
		return fmt.Errorf("error preparing query: %w", err)
	}
	stmt.BindText(1, visitorId)
	stmt.BindText(2, videoId)
	if _, err = stmt.Step(); err != nil {
		return err
	}
	err = stmt.Reset()
	if err != nil {
		return err
	}

	return nil
}

func (s *Router) GetRandom(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)

	visitor := r.Header.Get("visitor")
	params := r.URL.Query()

	conn := s.db.Get(context.Background())
	defer s.db.Put(conn)

	searchCriteria := ParseQueryParams(params)
	video, err := s.TakeFirstUnseen(conn, searchCriteria, visitor)
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
		err = s.RememberSeen(conn, visitor, video.ID)
		if err != nil {
			log.Println("error remembering seen:", err.Error())
		}
		return
	}
}
