package youtube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type YouTubeRequester struct {
	noRedirectClient *http.Client
	token             string
	baseUrl          string
}

func NewYouTubeRequester(token string) *YouTubeRequester {
	return &YouTubeRequester{
		baseUrl: "https://www.googleapis.com/youtube/v3",
		token:    token,
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
	req.URL.RawQuery += fmt.Sprintf("id=%s&part=statistics,snippet", strings.Join(ids, ","))

	res, err := y.Request(req)
	if err != nil {
		return nil, err
	}

	r := new(VideosResponse)
	json.NewDecoder(res.Body).Decode(r)

	return r, nil
}

func (y *YouTubeRequester) IsShort(videoID string, uploadDate int64) (bool, error) {
	
	// youtube shorts were released in 14.09.2020
	if uploadDate < 1600041600 {
		return false, nil
	}

	res, err := y.noRedirectClient.Head(fmt.Sprintf("https://www.youtube.com/shorts/%s", videoID))
	if err != nil {
		return false, err
	}
	return res.StatusCode == 200, nil
}
