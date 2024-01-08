package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
	"ytstalker/backend/youtube"

	"github.com/gorilla/mux"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func (s *Router) GetVideo(w http.ResponseWriter, r *http.Request) {
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
		ID:         stmt.GetText("id"),
		UploadedAt: stmt.GetInt64("uploaded"),
		Title:      stmt.GetText("title"),
		Views:      stmt.GetInt64("views"),
		Vertical:   stmt.GetBool("vertical"),
		Category:   stmt.GetInt64("category"),
	}
	err = stmt.Reset()
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	res.Reactions, _ = GetReactionStats(conn, res.Video.ID)
	encoder.Encode(res)
}

func (s *Router) GetRandom(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)

	visitor := r.Header.Get("visitor")
	params := r.URL.Query()

	conn := s.db.Get(r.Context())
	defer s.db.Put(conn)

	searchCriteria := ParseQueryParams(params)
	video, err := s.TakeFirstUnseen(conn, searchCriteria, visitor)
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
	var found bool
	for !found {
		results := make(map[string]*Video)

		searchResult, err := s.ytr.Search("inurl:" + youtube.RandomYoutubeVideoId())
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			encoder.Encode(Message{"couldn't find video"})
			return
		}
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

		ids := make([]string, len(results))
		for _, video := range results {
			ids = append(ids, video.ID)
		}

		videoInfoResult, err := s.ytr.VideosInfo(ids)
		if err != nil {
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

		// clear out fucking gaming videos trash i hate it
		for videoId, video := range results {
			if video.Category == 20 {
				delete(results, videoId)
			}
		}

		for _, video := range results {
			if searchCriteria.CheckVideo(video) {
				response.Video = video
				encoder.Encode(response)
				found = true
				err = s.RememberSeen(conn, visitor, video.ID)
				if err != nil {
					log.Println("error remembering seen:", err.Error())
				}
				break
			}
		}
		
		err = s.StoreVideos(conn, results)
		if err != nil {
			log.Println("couldn't store found videos", err.Error())
		}
		log.Println(len(results), "found videos stored")
	}
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

func (s *Router) StoreVideos(conn *sqlite.Conn, videos map[string]*Video) error {

	endFn, err := sqlitex.ImmediateTransaction(conn)
	if err != nil {
		return err
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

		if _, err := stmt.Step(); err != nil {
			return fmt.Errorf("%s %w", video.ID, err)
		}
		if err := stmt.Reset(); err != nil {
			return fmt.Errorf("error resetting stmt: %w", err)
		}
	}
	return nil
}

func (s *Router) RememberSeen(conn *sqlite.Conn, visitorId string, videoId string) error {

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
