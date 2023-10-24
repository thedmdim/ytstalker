package api

import (
	"net/url"
	"strconv"
	"strings"
)

type Message struct {
	Detail string `json:"detail"`
}

type Video struct {
	Id string `json:"id"`
	Uploaded int64 `json:"uploaded"`
	Title string `json:"title"`
	Views int `json:"views"`
	Vertical bool `json:"vertical"`
	Category int `json:"category"`
}

type SearchCriteria struct {
	Visitor string
	ViewsFrom string
	ViewsTo string
	YearsFrom string
	YearsTo string
	Category string
	Horizonly bool
}

func ParseQueryParams(params url.Values) *SearchCriteria {

	sc := &SearchCriteria{}

	viewsValues := strings.Split(params.Get("views"), "-")
	if len(viewsValues) == 2 {
		sc.ViewsFrom = viewsValues[0]
		sc.ViewsTo = viewsValues[1]
	}

	yearsValues := strings.Split(params.Get("years"), "-")
	if len(viewsValues) != 2 {
		sc.YearsFrom = yearsValues[0]
		sc.YearsTo = yearsValues[1]
	}

	category := params.Get("category")
	_, err := strconv.ParseBool(params.Get("category"))
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
		conditions = append(conditions, "views >= " + sc.ViewsFrom + " AND")
	}
	if _, err := strconv.Atoi(sc.ViewsTo); err == nil {
		conditions = append(conditions, "views <= " + sc.ViewsTo + " AND")
	}
	if _, err := strconv.Atoi(sc.YearsFrom); err == nil {
		conditions = append(conditions, "uploaded >= " + sc.YearsFrom + " AND")
	}
	if _, err := strconv.Atoi(sc.YearsFrom); err == nil {
		conditions = append(conditions, "uploaded <= " + sc.YearsTo + " AND")
	}
	if sc.Horizonly {
		conditions = append(conditions, "vertical = 0")
	}
	if len(conditions) > 1 {
		return strings.Join(conditions, " ")
	}
	return ""
}