package api

import (
	"encoding/json"
	"fmt"
	"go-youtube-stalker-site/backend/youtube"
	"log"
	"net/http"
	"time"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func (s *Server) RandomHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)

	params := r.URL.Query()
	if params.Get("visitor") == "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		encoder.Encode(Message{"visitor param required"})
		return
	}

	conn := s.db.Get(r.Context())
	defer s.db.Put(conn)

	video, err := s.TakeFirstUnseen(conn, ParseQueryParams(params))
	if err != nil {
		log.Println("take first unseen failed:", err.Error())
	}

	if video != nil {
		encoder.Encode(video)
		s.RememberSeen(conn, params.Get("visitor"), video.Id)
		return
	}
	log.Println("video in db not found, ask youtube api")

	// go ask youtube api for random video
	results := make(map[string]*Video)
	for {
		searchResult := s.ytr.Search("inurl:" + youtube.RandomYoutubeVideoId())
		if searchResult == nil {
			continue
		}
		nItems := len(searchResult.Items)
		if nItems == 0 {
			continue
		}
		log.Printf("%d videos found", nItems)

		for _, item := range searchResult.Items {
			video := Video{}
			video.Id = item.Id.VideoId
			video.Title = item.Snippet.Title
			parsed, err := time.Parse(time.RFC3339, item.Snippet.PublishedAt)
			if err == nil {
				video.Uploaded = parsed.Unix()
			}
			results[item.Id.VideoId] = &video
		}
		break
	}

	ids := make([]string, len(results))
	for _, video := range results {
		ids = append(ids, video.Id)
	}
	
	videoInfoResult := s.ytr.VideosInfo(ids)
	if videoInfoResult == nil {
		w.WriteHeader(http.StatusBadGateway)
		encoder.Encode(Message{"couldn't find video"})
		return
	}
	for _, item := range videoInfoResult.Items {
		video := results[item.Id]
		video.Category = item.Snippet.CategoryId
		video.Views = item.Statistics.ViewCount
	}

	for videoId, video := range results {
		short, err := s.ytr.IsShort(videoId)
		if err != nil {
			log.Println("error defining short:", err.Error())
		}
		video.Vertical = short
	}

	for _, video := range results {
		err = s.RememberSeen(conn, params.Get("visitor"), video.Id)
		if err == nil {
			encoder.Encode(video)
			break
		}
		log.Println(err.Error())
	}

	errs := s.StoreVideos(conn, results)
	if len(errs) > 0 {
		log.Println("[couldn't store found videos]")
		for _, e := range errs {
			log.Println(e.Error())
		}
	}
}

func (s *Server) TakeFirstUnseen(conn *sqlite.Conn, sc *SearchCriteria) (*Video, error) {

	query := fmt.Sprintf(
		`
		SELECT id, uploaded, title, views, vertical, category
		FROM videos
		WHERE id NOT IN (
			SELECT videos_visitors.video_id
			FROM videos_visitors
			WHERE videos_visitors.visitor_id = %s
		) %s
		LIMIT 1
		`,
		sc.Visitor,
		sc.MakeWhere(),
	)

	video := &Video{}

	stmt, _, err := conn.PrepareTransient(query)
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

	video.Id = stmt.GetText("id")
	video.Uploaded = stmt.GetInt64("uploaded")
	video.Title = stmt.GetText("title")
	video.Views = int(stmt.GetInt64("views"))
	video.Vertical = stmt.GetBool("vertical")
	video.Category = int(stmt.GetInt64("category"))

	return video, nil
}

func (s *Server) StoreVideos(conn *sqlite.Conn, videos map[string]*Video) []error {

	var errs []error

	endFn, err := sqlitex.ImmediateTransaction(conn)
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	defer endFn(&err)

	stmt := conn.Prep("INSERT INTO videos (id, uploaded, title, views, vertical, category) VALUES (?, ?, ?, ?, ?, ?);")
	for _, video := range videos {
		
		stmt.SetText("id", video.Id)
		stmt.SetInt64("uploaded", video.Uploaded)
		stmt.SetText("title", video.Title)
		stmt.SetInt64("views", int64(video.Views))
		stmt.SetBool("vertical", video.Vertical)
		stmt.SetInt64("category", int64(video.Views))

		if _, e := stmt.Step(); e != nil {
			errs = append(errs, fmt.Errorf("%s %w", video.Id, e))
		}
		if err := stmt.Reset(); err != nil {
			errs = append(errs, fmt.Errorf("error resetting stmt: %w", err))
			return errs
		}
	}
	return nil
}

func (s *Server) RememberSeen(conn *sqlite.Conn, visitorId string, videoId string) error {

	endFn, err := sqlitex.ImmediateTransaction(conn)
	if err != nil {
		return err
	}
	defer endFn(&err)

	now := time.Now().Unix()
	stmt, err := conn.Prepare(`
		INSERT INTO visitors (id, last_seen)
		VALUES(?, ?)
		ON CONFLICT (id)
		DO UPDATE SET last_seen=?;`)
	if err != nil {
		return fmt.Errorf("error preparing query: %w", err)
	}
	stmt.BindText(1, visitorId)
	stmt.BindInt64(2, now)
	stmt.BindInt64(3, now)
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

	return nil
}