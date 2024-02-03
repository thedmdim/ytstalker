package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type Message struct {
	Detail string `json:"detail"`
}

type CamWithReactions struct {
	Cam       *Cam      `json:"cam,omitempty"`
	Reactions Reactions `json:"reactions"`
}

type Cam struct {
	ID           string `json:"id"`
	Addr         string `json:"addr"`
	Adminka      string `json:"adminka"`
	Stream       string `json:"stream"`
	Manufacturer string `json:"manufacturer"`
	Country      string `json:"country"`
}

type Reactions struct {
	Cools   int64 `json:"cools"`
	Trashes int64 `json:"trashes"`
}

type SearchCriteria struct {
	Country      string
	Manufacturer string
}

func ParseQueryParams(params url.Values) *SearchCriteria {

	sc := &SearchCriteria{}

	sc.Manufacturer = params.Get("manufacturer")
	sc.Country = params.Get("country")

	return sc
}

func (sc *SearchCriteria) MakeWhere() string {
	var conditions []string

	if sc.Manufacturer != "" {
		conditions = append(conditions, "country = "+sc.Manufacturer)
	}

	if sc.Country != "" {
		conditions = append(conditions, "country = "+sc.Country)
	}

	if len(conditions) > 0 {
		return "AND " + strings.Join(conditions, " AND ")
	}
	return ""
}

func (s *Router) TakeFirstUnseen(conn *sqlite.Conn, sc *SearchCriteria, visitor string) (*Cam, error) {

	camera := &Cam{}

	stmt, _, err := conn.PrepareTransient(fmt.Sprintf(`
		SELECT id, uploaded, title, views, vertical, category
		FROM videos
		WHERE id NOT IN (
			SELECT cams_visitors.cam_id
			FROM cams_visitors
			WHERE cams_visitors.visitor_id = %s
		) %s
		ORDER BY random()
		LIMIT 1`,
		visitor,
		sc.MakeWhere(),
	))
	if err != nil {
		return nil, fmt.Errorf("error preparing query: %w", err)
	}
	row, err := stmt.Step()
	if err != nil {
		// TO DO : if not row, select seen
		return nil, err
	}
	if !row {
		return nil, nil
	}

	camera.ID = stmt.GetText("id")
	camera.Addr = stmt.GetText("addr")
	camera.Adminka = stmt.GetText("addr")
	camera.Stream = stmt.GetText("stream")
	camera.Manufacturer = stmt.GetText("manufacturer")
	camera.Country = stmt.GetText("country")

	err = stmt.Reset()
	if err != nil {
		return camera, err
	}

	return camera, nil
}

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

	stmt, err = conn.Prepare(`INSERT INTO videos_visitors (visitor_id, cam_id) VALUES (?, ?);`)
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

	response := &CamWithReactions{}

	if video != nil {
		reactions, err := GetReaction(conn, video.ID)
		if err != nil {
			log.Println("couldn't save reaction:", err.Error())
		}

		response.Cam = video
		response.Reactions = reactions

		encoder.Encode(response)
		err = s.RememberSeen(conn, visitor, video.ID)
		if err != nil {
			log.Println("error remembering seen:", err.Error())
		}
		return
	}
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
