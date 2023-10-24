package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-youtube-stalker-site/backend/youtube"
	"log"
	"net/http"
	"strings"
	"time"
)

var ErrorInvalidQueryParam = errors.New("invalid query param")

func (s *Server) Random(w http.ResponseWriter, r *http.Request) {
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

	err = s.StoreVideos(results)
	if err != nil {
		log.Println("couldn't store found videos:", err.Error())
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

	stmt := fmt.Sprintf(`
		SELECT id, uploaded, title, views, vertical, category
		FROM videos
		WHERE %s
		AND videos.id NOT IN (
			SELECT videos_visitors.video_id
			FROM videos_visitors
			WHERE videos_visitors.visitor_id = %s
		)
	`, sc.MakeWhere(), sc.Visitor)

	video := &Video{}
	err := s.db.QueryRow(stmt).Scan(video.Id, video.Uploaded, video.Title, video.Views, video.Vertical, video.Category)
	return video, err

}

func (s *Server) StoreVideos(videos map[string]*Video) error {
	stmtParts := []string{"INSERT INTO videos (id, uploaded, title, views, vertical, category) VALUES"}
	for _, video := range videos {
		stmtParts = append(stmtParts, fmt.Sprintf("(%s, %d, %s, %d, %t, %d)", video.Id, video.Uploaded, video.Title, video.Views, video.Vertical, video.Category))
	}
	stmt := strings.Join(stmtParts, ", ") + ";"
	s.wlock.Lock()
	_, err := s.db.Exec(stmt)
	s.wlock.Unlock()
	return err
}

func (s *Server) RememberSeen(visitorId string, videoId string) error {

	const stmt = `
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
	now := time.Now().Unix()
	s.wlock.Lock()
	_, err := s.db.Exec(stmt, visitorId, now, now, videoId)
	s.wlock.Unlock()
	return err
}
