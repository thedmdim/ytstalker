package api

import (
	"html/template"
	"log"
	"net/http"
	"strings"
)

// Parse and execute the templates
var templates = template.Must(template.ParseGlob("frontend/*/*.html"))

func (s *Server) PagesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("get / request")
	// Check the request URL
	if r.URL.Path == "/" {
		// Execute the index template
		err := templates.ExecuteTemplate(w, "index.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Handle other routes, e.g., '/about'
		templateName := strings.TrimLeft(r.URL.Path, "/")
		err := templates.ExecuteTemplate(w, templateName + ".html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
	}
}