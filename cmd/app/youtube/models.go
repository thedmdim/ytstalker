package youtube

// models for search endpoint
type SearchResponse struct {
	Items    []SearchItem   `json:"items"`
	PageInfo SearchPageInfo `json:"pageInfo"`
}

type SearchPageInfo struct {
	TotalResults int `json:"totalResults"`
}

type SearchItem struct {
	Id      SearchId      `json:"id"`
	Snippet SearchSnippet `json:"snippet"`
}

type SearchId struct {
	VideoId string `json:"videoId"`
}

type SearchSnippet struct {
	PublishedAt string `json:"publishedAt"`
	Title       string `json:"title"`
}

// models for videos endpoint
type VideosResponse struct {
	Items []VideosItem `json:"items"`
}

type VideosItem struct {
	Id         string           `json:"id"`
	Statistics VideosStatistics `json:"statistics"`
	Snippet    VideosSnippet    `json:"snippet"`
}

type VideosStatistics struct {
	ViewCount string `json:"viewCount"`
}

type VideosSnippet struct {
	CategoryId string `json:"categoryId"`
}
