package app

import (
	"net/http"
	"strings"
)

func (s *Server) routeStaticFiles(w http.ResponseWriter, r *http.Request, path string) bool {
	if strings.HasPrefix(path, "/static/") {
		rel := strings.TrimPrefix(path, "/static/")
		if strings.HasPrefix(rel, "pages/") {
			http.NotFound(w, r)
			return true
		}
		if strings.HasSuffix(rel, ".html") && !strings.Contains(rel, "/") {
			s.serveStaticHTML(w, r, rel)
			return true
		}
		s.serveFileRoot(w, r, s.cfg.StaticDir, strings.TrimPrefix(path, "/static/"))
		return true
	}
	if strings.HasPrefix(path, "/uploads/") {
		s.serveFileRoot(w, r, s.cfg.UploadDir, strings.TrimPrefix(path, "/uploads/"))
		return true
	}
	return false
}
