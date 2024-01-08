package youtube

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

var ErrorApiQuota = errors.New("YouTube API quota exceeded")
var ErrorMaxTries = errors.New("you've reached max api tries limit")
const YtApiMaxTriesLimit = 100

func (y *YouTubeRequester) Request(req *http.Request) (*http.Response, error) {
	// Just wrap http.Get to add http code errors
	// retries with fresh api keys if provided

	for i := 1; i < YtApiMaxTriesLimit; i++ {
		q := req.URL.Query()
		q.Add("key", y.conf.YouTubeApiKeys[y.currentApiKeyN])
		req.URL.RawQuery = q.Encode()
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		if res.StatusCode == 403 {
			log.Println("YouTube API quota exceeded, I'll try to use another API key..")
			if len(y.conf.YouTubeApiKeys) > y.currentApiKeyN+1 {
				y.currentApiKeyN++
				continue
			}
			return nil, ErrorApiQuota
		}
		if res.StatusCode != 200 {
			return nil, fmt.Errorf("%d %s", res.StatusCode, req.URL.String())
		}
		return res, err
	}
	return nil, ErrorMaxTries
}

const base64range string = "0123456789abcdefghijklmnopqrstuvwxyz-_"

func RandomYoutubeVideoId() string {
	/*
		we don't need uppercase and downcase both presented
		because api search isn't case sensetive
	*/

	var id []byte

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 5; i++ {
		id = append(id, base64range[random.Intn(37)])
	}

	return string(id)
}
