package app

import (
	"net/http"
	"strings"
)

// routeUserFrontend serves the blog/community user-facing pages.
func (s *Server) routeUserFrontend(w http.ResponseWriter, r *http.Request, path string) bool {
	if r.Method != http.MethodGet {
		return false
	}
	switch path {
	case "/user-login":
		s.serveStaticHTML(w, r, "user_login.html")
	case "/user-register":
		s.serveStaticHTML(w, r, "user_register.html")
	case "/blog":
		s.serveStaticHTML(w, r, "blog.html")
	case "/editor":
		s.serveStaticHTML(w, r, "editor.html")
	case "/account":
		s.serveStaticHTML(w, r, "account.html")
	case "/messages":
		s.serveStaticHTML(w, r, "messages.html")
	case "/groups":
		s.serveStaticHTML(w, r, "groups.html")
	default:
		if strings.HasPrefix(path, "/blog/") {
			s.serveStaticHTML(w, r, "article.html")
			return true
		}
		if strings.HasPrefix(path, "/space/") {
			s.serveStaticHTML(w, r, "space.html")
			return true
		}
		return false
	}
	return true
}
