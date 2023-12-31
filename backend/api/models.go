package api

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Message struct {
	Detail string `json:"detail"`
}

type RandomResponse struct {
	Video     *Video     `json:"video,omitempty"`
	Reactions Reactions `json:"reactions"`
}

type Video struct {
	ID         string `json:"id"`
	UploadedAt int64  `json:"uploaded"`
	Title      string `json:"title"`
	Views      int64  `json:"views"`
	Vertical   bool   `json:"vertical"`
	Category   int64  `json:"category"`
}

func (v *Video) GetFormattedUploadDate(timestamp int64) string {
	t := time.Unix(v.UploadedAt, 0)
	return t.Format("02.01.06")
}

type Reactions struct {
	Cools   int64 `json:"cools"`
	Trashes int64 `json:"trashes"`
}

type SearchCriteria struct {
	ViewsFrom string
	ViewsTo string
	YearsFrom string
	YearsTo string
	Category string
	Horizonly bool
	Musiconly bool
}

func ParseQueryParams(params url.Values) *SearchCriteria {

	sc := &SearchCriteria{}

	viewsValues := strings.Split(params.Get("views"), "-")
	if len(viewsValues) == 2 {
		sc.ViewsFrom = viewsValues[0]
		sc.ViewsTo = viewsValues[1]
	}

	yearsValues := strings.Split(params.Get("years"), "-")
	if len(viewsValues) == 2 {
		sc.YearsFrom = yearsValues[0]
		sc.YearsTo = yearsValues[1]
	}

	category := params.Get("category")
	_, err := strconv.Atoi(category)
	if err == nil {
		sc.Category = category
	}

	horizonly, err := strconv.ParseBool(params.Get("horizonly"))
	if err == nil {
		sc.Horizonly = horizonly
	}

	return sc
}

func (sc *SearchCriteria) MakeWhere() string {
	var conditions []string

	if _, err := strconv.Atoi(sc.ViewsFrom); err == nil {
		conditions = append(conditions, "views >= "+sc.ViewsFrom)
	}
	if _, err := strconv.Atoi(sc.ViewsTo); err == nil {
		conditions = append(conditions, "views <= "+sc.ViewsTo)
	}
	if year, err := strconv.Atoi(sc.YearsFrom); err == nil {
		timestamp := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
		conditions = append(conditions, fmt.Sprintf("uploaded >= %d", timestamp))
	}
	if year, err := strconv.Atoi(sc.YearsTo); err == nil {
		timestamp := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
		conditions = append(conditions, fmt.Sprintf("uploaded <= %d", timestamp))
	}
	if sc.Horizonly {
		conditions = append(conditions, "vertical = 0")
	}
	if sc.Category != "" {
		conditions = append(conditions, "category = " + sc.Category)
	}
	if len(conditions) > 0 {
		return "AND " + strings.Join(conditions, " AND ")
	}
	return ""
}

func (sc *SearchCriteria) CheckVideo(video *Video) bool {
	if viewsFrom, err := strconv.ParseInt(sc.ViewsFrom, 10, 64); err == nil && video.Views < viewsFrom {
		return false
	}
	if viewsTo, err := strconv.ParseInt(sc.ViewsTo, 10, 64); err == nil && video.Views > viewsTo {
		return false
	}
	if yearFrom, err := strconv.Atoi(sc.YearsFrom); err == nil {
		if video.UploadedAt < time.Date(yearFrom, time.January, 1, 0, 0, 0, 0, time.UTC).Unix() {
			return false
		}
	}
	if yearTo, err := strconv.Atoi(sc.YearsTo); err == nil {
		if video.UploadedAt > time.Date(yearTo, time.January, 1, 0, 0, 0, 0, time.UTC).Unix() {
			return false
		}
	}
	if sc.Horizonly {
		if video.Vertical {
			return false
		}
	}
	if sc.Musiconly {
		if video.Category != 10 {
			return false
		}
	}
	return true
}
