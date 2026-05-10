package app

import (
	"net/http"
	"strings"
)

// routePublicAPI exposes anonymous read-only/promotional APIs.
func (s *Server) routePublicAPI(w http.ResponseWriter, r *http.Request, path string) bool {
	switch {
	case path == "/api/status" && r.Method == http.MethodGet:
		s.handleStatus(w, r)
	case path == "/api/content" && r.Method == http.MethodGet:
		s.handleGetContent(w, r)
	case strings.HasPrefix(path, "/api/news/") && r.Method == http.MethodGet:
		s.handleGetNews(w, r, pathValue(path, "/api/news/"))
	case strings.HasPrefix(path, "/api/review/albums/") && r.Method == http.MethodGet:
		s.handleGetReviewAlbum(w, r, pathValue(path, "/api/review/albums/"))
	default:
		return false
	}
	return true
}
