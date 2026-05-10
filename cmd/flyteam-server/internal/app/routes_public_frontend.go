package app

import (
	"net/http"
	"strings"
)

// routePublicFrontend serves the promotional site seen by anonymous visitors.
func (s *Server) routePublicFrontend(w http.ResponseWriter, r *http.Request, path string) bool {
	if r.Method != http.MethodGet {
		return false
	}
	switch path {
	case "/":
		s.serveStaticHTML(w, r, "index.html")
	case "/flyteamers":
		s.serveStaticHTML(w, r, "flyteamers.html")
	case "/recruit":
		s.serveStaticHTML(w, r, "recruit.html")
	case "/news":
		s.serveStaticHTML(w, r, "news.html")
	case "/awards":
		s.serveStaticHTML(w, r, "awards.html")
	case "/review":
		s.serveStaticHTML(w, r, "review.html")
	case "/intro":
		s.serveStaticHTML(w, r, "intro.html")
	default:
		if strings.HasPrefix(path, "/review/") {
			s.serveStaticHTML(w, r, "review_detail.html")
			return true
		}
		return false
	}
	return true
}
