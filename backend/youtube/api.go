package youtube

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"go-youtube-stalker-site/backend/conf"
)

type YouTubeRequester struct {
	client *http.Client 
	conf *conf.Config
	currentApiKeyN int
}

func NewYouTubeRequester(conf *conf.Config) *YouTubeRequester {
	return &YouTubeRequester{
		conf: conf,
		client: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// "inurl:" + RandomYoutubeVideoId()
func (y *YouTubeRequester) Search(query string) *SearchResponse {


	req, _ := http.NewRequest("GET", y.conf.YouTubeApiUrl + "/search", nil)
	q := url.Values{}
    q.Add("part", "snippet")
    q.Add("maxResults", "50")
	q.Add("type", "video")
	q.Add("q", query)
	req.URL.RawQuery = q.Encode()	

	res, err := y.Request(req)
	if err != nil {
		log.Println(err.Error())
		return nil
	}

	r := new(SearchResponse)
	json.NewDecoder(res.Body).Decode(r)

	return r
}

func (y *YouTubeRequester) VideosInfo(ids []string) *VideosResponse {

	req, _ := http.NewRequest("GET", y.conf.YouTubeApiUrl + "/videos", nil)
	q := url.Values{}
    q.Add("part", "statistics,snippet")
	q.Add("id", strings.Join(ids, ","))

	res, err := y.Request(req)
	if err != nil {
		log.Println(err.Error())
		return nil
	}

	r := new(VideosResponse)
	json.NewDecoder(res.Body).Decode(r)

	return r
}

func(y *YouTubeRequester) IsShort(id string) (bool, error) {
	res, err := y.client.Head(fmt.Sprintf("https://www.youtube.com/shorts/%s", id))
	if err != nil {
		return false, err
	}
	return res.StatusCode == 200, nil
}