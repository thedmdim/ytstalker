package api

import (
	"encoding/json"
	"fmt"
	"go-youtube-stalker-site/backend/youtube"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func (s *Server) GetVideo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)

	vars := mux.Vars(r)

	conn := s.db.Get(r.Context())
	defer s.db.Put(conn)

	res := &RandomResponse{}

	stmt := conn.Prep(`
		SELECT id, uploaded, title, views, vertical, category
		FROM videos
		WHERE id = ?`)

	stmt.BindText(1, vars["video_id"])
	rows, err := stmt.Step()
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	if !rows {
		w.WriteHeader(http.StatusNotFound)
		encoder.Encode(Message{"couldn't find video"})
		return
	}
	res.Video = &Video{
		ID: stmt.GetText("id"),
		UploadedAt: stmt.GetInt64("uploaded"),
		Title: stmt.GetText("title"),
		Views: stmt.GetInt64("views"),
		Vertical: stmt.GetBool("vertical"),
		Category: stmt.GetInt64("category"),
	}
	err = stmt.Reset()
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	res.Reactions, _ = GetReactionStats(conn, res.Video.ID)
	encoder.Encode(res)
}

func (s *Server) GetRandom(w http.ResponseWriter, r *http.Request) {

	log.Println("get random")

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)

	visitor := r.Header.Get("visitor")
	params := r.URL.Query()

	conn := s.db.Get(r.Context())
	defer s.db.Put(conn)

	video, err := s.TakeFirstUnseen(conn, ParseQueryParams(params), visitor)
	if err != nil {
		log.Println("take first unseen failed:", err.Error())
	}

	response := &RandomResponse{}

	if video != nil {
		reactions, err := GetReactionStats(conn, video.ID)
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
			video.ID = item.Id.VideoId
			video.Title = item.Snippet.Title
			parsed, err := time.Parse(time.RFC3339, item.Snippet.PublishedAt)
			if err == nil {
				video.UploadedAt = parsed.Unix()
			}
			results[item.Id.VideoId] = &video
		}
		break
	}

	ids := make([]string, len(results))
	for _, video := range results {
		ids = append(ids, video.ID)
	}

	videoInfoResult := s.ytr.VideosInfo(ids)
	if videoInfoResult == nil {
		w.WriteHeader(http.StatusBadGateway)
		encoder.Encode(Message{"couldn't find video"})
		return
	}
	for _, item := range videoInfoResult.Items {
		video := results[item.Id]
		video.Category, _ = strconv.ParseInt(item.Snippet.CategoryId, 10, 64)
		video.Views, _ = strconv.ParseInt(item.Statistics.ViewCount, 10, 64)
	}

	for videoId, video := range results {
		short, err := s.ytr.IsShort(videoId)
		if err != nil {
			log.Println("error defining short:", err.Error())
		}
		video.Vertical = short
	}

	for _, video := range results {
		err = s.RememberSeen(conn, visitor, video.ID)
		if err == nil {
			response.Video = video
			encoder.Encode(response)
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

func (s *Server) TakeFirstUnseen(conn *sqlite.Conn, sc *SearchCriteria, visitor string) (*Video, error) {

	video := &Video{}

	stmt, _, err := conn.PrepareTransient(fmt.Sprintf(`
		SELECT id, uploaded, title, views, vertical, category
		FROM videos
		WHERE id NOT IN (
			SELECT videos_visitors.video_id
			FROM videos_visitors
			WHERE videos_visitors.visitor_id = %s
		) %s
		LIMIT 1`,
		visitor,
		sc.MakeWhere(),
	))
	if err != nil {
		return nil, fmt.Errorf("error preparing query: %w", err)
	}
	rows, err := stmt.Step()
	log.Println(rows)
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

func (s *Server) StoreVideos(conn *sqlite.Conn, videos map[string]*Video) []error {
	log.Println("store found videos", len(videos))
	var errs []error

	endFn, err := sqlitex.ImmediateTransaction(conn)
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	defer endFn(&err)

	stmt := conn.Prep("INSERT INTO videos (id, uploaded, title, views, vertical, category) VALUES (?, ?, ?, ?, ?, ?);")
	for _, video := range videos {

		stmt.BindText(1, video.ID)
		stmt.BindInt64(2, video.UploadedAt)
		stmt.BindText(3, video.Title)
		stmt.BindInt64(4, int64(video.Views))
		stmt.BindBool(5, video.Vertical)
		stmt.BindInt64(6, int64(video.Category))

		if _, e := stmt.Step(); e != nil {
			errs = append(errs, fmt.Errorf("%s %w", video.ID, e))
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
