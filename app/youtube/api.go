package youtube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"ytstalker/app/conf"
)

type YouTubeRequester struct {
	noRedirectClient *http.Client
	conf             *conf.Config
	currentApiKeyN   int
	baseUrl          string
}

func NewYouTubeRequester(conf *conf.Config) *YouTubeRequester {
	return &YouTubeRequester{
		baseUrl: "https://www.googleapis.com/youtube/v3",
		conf:    conf,
		noRedirectClient: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// "inurl:" + RandomYoutubeVideoId()
func (y *YouTubeRequester) Search(query string) (*SearchResponse, error) {

	req, _ := http.NewRequest("GET", y.baseUrl+"/search", nil)
	q := url.Values{}
	q.Add("part", "snippet")
	q.Add("maxResults", "50")
	q.Add("type", "video")
	q.Add("q", query)
	req.URL.RawQuery = q.Encode()

	res, err := y.Request(req)
	if err != nil {
		return nil, err
	}

	r := new(SearchResponse)
	json.NewDecoder(res.Body).Decode(r)

	return r, nil
}

func (y *YouTubeRequester) VideosInfo(ids []string) (*VideosResponse, error) {

	req, _ := http.NewRequest("GET", y.baseUrl+"/videos", nil)
	// q := url.Values{}
	// q.Add("part", "statistics,snippet")
	// q.Add("id", strings.Join(ids, ","))
	req.URL.RawQuery += fmt.Sprintf("id=%s&part=statistics,snippet", strings.Join(ids, ","))

	res, err := y.Request(req)
	if err != nil {
		return nil, err
	}

	r := new(VideosResponse)
	json.NewDecoder(res.Body).Decode(r)

	return r, nil
}

func (y *YouTubeRequester) IsShort(id string) (bool, error) {
	res, err := y.noRedirectClient.Head(fmt.Sprintf("https://www.youtube.com/shorts/%s", id))
	if err != nil {
		return false, err
	}
	return res.StatusCode == 200, nil
}
