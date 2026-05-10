package app

import (
	"errors"
	"net/http"
	"strings"
)

// routeAdminBackendPage serves the administrator login and console pages.
func (s *Server) routeAdminBackendPage(w http.ResponseWriter, r *http.Request, path string) bool {
	if r.Method != http.MethodGet {
		return false
	}
	switch path {
	case "/login":
		s.handleLoginPage(w, r)
	case "/admin":
		s.handleAdminPage(w, r)
	default:
		return false
	}
	return true
}

// routeAdminBackendAPI contains administrator, super administrator, and audit APIs.
func (s *Server) routeAdminBackendAPI(w http.ResponseWriter, r *http.Request, path string) bool {
	if s.routeAdminBlogOps(w, r, path) {
		return true
	}
	switch {
	case path == "/api/admin/login" && r.Method == http.MethodPost:
		s.handleAdminLogin(w, r)
	case path == "/api/admin/logout" && r.Method == http.MethodPost:
		s.handleAdminLogout(w, r)
	case path == "/api/admin/ping" && r.Method == http.MethodGet:
		s.handleAdminPing(w, r)
	case path == "/api/admin/users" && r.Method == http.MethodGet:
		s.handleAdminUsers(w, r)
	case path == "/api/admin/users" && r.Method == http.MethodPost:
		s.handleAddAdminUser(w, r)
	case strings.HasPrefix(path, "/api/admin/users/") && strings.HasSuffix(path, "/password") && r.Method == http.MethodPut:
		s.handleUpdateAdminPassword(w, r, strings.TrimSuffix(pathValue(path, "/api/admin/users/"), "/password"))
	case strings.HasPrefix(path, "/api/admin/users/") && strings.HasSuffix(path, "/role") && r.Method == http.MethodPut:
		s.handleUpdateAdminRole(w, r, strings.TrimSuffix(pathValue(path, "/api/admin/users/"), "/role"))
	case strings.HasPrefix(path, "/api/admin/users/") && r.Method == http.MethodDelete:
		s.handleDeleteAdminUser(w, r, pathValue(path, "/api/admin/users/"))
	default:
		return false
	}
	return true
}

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.adminFromRequest(r); ok {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}
	s.serveStaticHTML(w, r, "login.html")
}

func (s *Server) handleAdminPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.adminFromRequest(r); !ok {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	s.serveStaticHTML(w, r, "admin.html")
}

func (s *Server) requiresSiteAdminAPI(path, method string) bool {
	if path == "/api/recruit/list" {
		return method == http.MethodGet
	}
	if path == "/api/recruit/stats" {
		return false
	}
	if strings.HasPrefix(path, "/api/recruit/") && path != "/api/recruit/apply" && path != "/api/recruit/captcha" && path != "/api/recruit/halls" && path != "/api/recruit/stats" {
		return true
	}
	if strings.HasPrefix(path, "/api/awards") || strings.HasPrefix(path, "/api/seniors") || strings.HasPrefix(path, "/api/review") {
		return isMutating(method)
	}
	if strings.HasPrefix(path, "/api/news") {
		return isMutating(method)
	}
	if strings.HasPrefix(path, "/api/content") {
		return isMutating(method)
	}
	if strings.HasPrefix(path, "/api/ingest") {
		return true
	}
	switch path {
	case "/api/upload", "/api/upload/images", "/api/upload/awards/images", "/api/upload/seniors/images", "/api/upload/review/images", "/api/upload/news/images":
		return true
	}
	return false
}

func (s *Server) requiresAdminCSRF(path string) bool {
	if path == "/api/admin/login" {
		return false
	}
	if strings.HasPrefix(path, "/api/recruit/") && path != "/api/recruit/apply" {
		return true
	}
	for _, p := range []string{"/api/admin", "/api/superadmin", "/api/awards", "/api/seniors", "/api/news", "/api/review", "/api/content", "/api/ingest", "/api/upload"} {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func (s *Server) checkCSRF(r *http.Request) error {
	cookie, err := r.Cookie("admin_session")
	if err != nil || cookie.Value == "" || r.Header.Get("X-Admin-Token") != "" {
		return nil
	}
	admin, ok := s.adminFromToken(cookie.Value)
	if !ok {
		return nil
	}
	if admin.CSRFToken == "" || r.Header.Get("X-CSRF-Token") != admin.CSRFToken {
		return errors.New("CSRF token missing or invalid.")
	}
	return nil
}
