package youtube

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var ErrorApiQuota = errors.New("YouTube API quota exceeded")
var ErrorMaxTries = errors.New("you've reached max api tries limit")

type Video struct {
	ID         string
	UploadedAt int64
	Title      string
	Views      int64
	Vertical   bool
	Category   int64
}


func (y *YouTubeRequester) Request(req *http.Request) (*http.Response, error) {

	q := req.URL.Query()
	q.Add("key", y.conf.YtApiKey)
	req.URL.RawQuery = q.Encode()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode == 403 {
		return nil, ErrorApiQuota

	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("%d %s", res.StatusCode, req.URL.String())
	}
	return res, err
}

func (y *YouTubeRequester) FindRandomVideos() (map[string]*Video, error) {
	results := make(map[string]*Video)

	var ids []string
	for i:=0; i < 100; i++ {
		ids = append(ids, "inurl:" + RandomYoutubeVideoId(6))
	}

	log.Println(ids)

	searchResult, err := y.Search(strings.Join(ids, " OR "))
	if err != nil {
		return nil, err
	}

	log.Println(searchResult)

	if searchResult == nil || len(searchResult.Items) == 0 {
		return results, nil
	}

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

	ids = nil
	for videoID := range results {
		ids = append(ids, videoID)
	}

	videoInfoResult, err := y.VideosInfo(ids)
	if err != nil {
		return nil, err
	}
	for _, item := range videoInfoResult.Items {
		video := results[item.Id]
		video.Category, _ = strconv.ParseInt(item.Snippet.CategoryId, 10, 64)
		video.Views, _ = strconv.ParseInt(item.Statistics.ViewCount, 10, 64)
	}

	for videoID, video := range results {
		short, err := y.IsShort(video.ID, video.UploadedAt)
		if err != nil {
			delete(results, videoID)
			log.Printf("error defining short (%s): %s", video.ID, err.Error())
		} else {
			video.Vertical = short
		}
	}

	// clear out fucking gaming videos trash i hate it
	for videoId, video := range results {
		if video.Category == 20 {
			delete(results, videoId)
		}
	}

	return results, nil
}



const base64range string = "0123456789abcdefghijklmnopqrstuvwxyz-_"

func RandomYoutubeVideoId(length int) string {
	/*
		we don't need uppercase and downcase both presented
		because api search isn't case sensetive
	*/

	var id []byte

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < length; i++ {
		id = append(id, base64range[random.Intn(37)])
	}

	return string(id)
}
