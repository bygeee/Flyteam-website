package app

import (
	"net/http"
	"strings"
)

// routeRecruitAPI keeps public recruitment submission and admin review routes together.
func (s *Server) routeRecruitAPI(w http.ResponseWriter, r *http.Request, path string) bool {
	switch {
	case path == "/api/recruit/captcha" && r.Method == http.MethodGet:
		s.handleRecruitCaptcha(w, r)
	case path == "/api/recruit/halls" && r.Method == http.MethodGet:
		s.handleRecruitHalls(w, r)
	case path == "/api/recruit/stats" && r.Method == http.MethodGet:
		s.handleRecruitStats(w, r)
	case path == "/api/recruit/apply" && r.Method == http.MethodPost:
		s.handleRecruitApply(w, r)
	case path == "/api/recruit/list" && r.Method == http.MethodGet:
		s.handleRecruitList(w, r)
	case strings.HasPrefix(path, "/api/recruit/") && r.Method == http.MethodPut:
		s.handleRecruitUpdate(w, r, pathValue(path, "/api/recruit/"))
	case strings.HasPrefix(path, "/api/recruit/") && r.Method == http.MethodDelete:
		s.handleRecruitDelete(w, r, pathValue(path, "/api/recruit/"))
	default:
		return false
	}
	return true
}
