package api

import (
	"context"
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

	video, err := s.TakeFirstUnseen(ParseQueryParams(params))
	if err != nil {
		log.Println("take first unseen failed:", err.Error())
	}
	if video != nil {
		encoder.Encode(video)
		s.RememberSeen(params.Get("visitor"), video.Id)
		return
	}

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

	errs := s.StoreVideos(results)
	if len(errs) != 0 {
		log.Println("[couldn't store found videos]")
		for _, err := range errs {
			log.Println(" - ", err.Error())
		}
	}

	for _, video := range results {
		err = s.RememberSeen(params.Get("visitor"), video.Id)
		if err != nil {
			log.Println(err.Error())
		}
		encoder.Encode(video)
		return
	}
}

func (s *Server) TakeFirstUnseen(sc *SearchCriteria) (*Video, error) {

	query := fmt.Sprintf(
		`
		SELECT id, uploaded, title, views, vertical, category
		FROM videos
		WHERE %s
		AND videos.id NOT IN (
			SELECT videos_visitors.video_id
			FROM videos_visitors
			WHERE videos_visitors.visitor_id = %s
		)
		LIMIT 1
		`,
		sc.MakeWhere(),
		sc.Visitor,
	)

	conn := s.db.Get(context.Background())
	defer s.db.Put(conn)

	video := &Video{}
	// video.Id, video.Uploaded, video.Title, video.Views, video.Vertical, video.Category

	err := sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			video.Id = stmt.GetText("id")
			video.Uploaded = stmt.GetInt64("uploaded")
			video.Title = stmt.GetText("title")
			video.Views = int(stmt.GetInt64("views"))
			video.Vertical = stmt.GetBool("vertical")
			video.Category = int(stmt.GetInt64("category"))
			return nil
		},
	})

	return video, err
}

func (s *Server) StoreVideos(videos map[string]*Video) []error {

	const query = "INSERT INTO videos (id, uploaded, title, views, vertical, category) VALUES (?, ?, ?, ?, ?, ?);"
	
	conn := s.db.Get(context.Background())
	defer s.db.Put(conn)

	var errs []error
	for _, video := range videos {
		options := &sqlitex.ExecOptions{Args: []any{video.Id, video.Uploaded, video.Title, video.Views, video.Vertical, video.Category}}
		err := sqlitex.Execute(conn, query, options)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s %w", video.Id, err))
		}
	}
	return errs
}

func (s *Server) RememberSeen(visitorId string, videoId string) error {

	const query = `
		INSERT INTO videos_visitors (visitor_id, video_id)
		VALUES (
			(
				INSERT INTO visitors (id, last_seen)
				VALUES (?, ?)
				ON CONFLICT (id) DO
				UPDATE SET last_seen=?
				RETURNING id
			),
			?
		);
	`
	conn := s.db.Get(context.Background())
	defer s.db.Put(conn)

	now := time.Now().Unix()
	err := sqlitex.Execute(
		conn,
		query,
		&sqlitex.ExecOptions{Args: []any{visitorId, now, now, videoId}},
	)

	return err
}
