package handlers

import "net/http"

type Rating struct {
	Best  []*RatedCam
	Worst []*RatedCam
}

type RatedCam struct {
	ID        string
	Title     string
	Reactions int64
}

const ratingLimit int64 = 10

func (s *Router) GetRating(w http.ResponseWriter, r *http.Request) {

	conn := s.db.Get(r.Context())
	defer s.db.Put(conn)

	stmt := conn.Prep(`
		SELECT SUM(reactions.like = ?) AS reactions_sum, reactions.cam_id
		FROM reactions
		JOIN cams ON cams.id = reactions.cam_id
		GROUP BY reactions.cam_id
		ORDER BY reactions_sum DESC
		LIMIT ?
	`)

	stmt.BindInt64(1, 1)
	stmt.BindInt64(2, ratingLimit)

	rating := &Rating{}
	for {
		row, err := stmt.Step()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if row {
			rating.Best = append(rating.Best, &RatedCam{
				ID:        stmt.GetText("cam_id"),
				Reactions: stmt.GetInt64("reactions_sum"),
			})
		} else { break }
	}

	stmt.Reset(); stmt.ClearBindings()

	stmt.BindInt64(1, 0)
	stmt.BindInt64(2, ratingLimit)

	for {
		row, err := stmt.Step()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if row {
			rating.Worst = append(rating.Worst, &RatedCam{
				ID:        stmt.GetText("cam_id"),
				Reactions: stmt.GetInt64("reactions_sum"),
			})
		} else { break }
	}

	templates.ExecuteTemplate(w, "rating.html", rating)

}