package app

import (
	"log"
	"net/http"
)

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("panic: %v", rec)
			writeError(w, http.StatusInternalServerError, "Internal server error.")
		}
	}()
	s.setSecurityHeaders(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	path := cleanPath(r.URL.Path)
	if isAdminStaticAsset(path) {
		if _, ok := s.adminFromRequest(r); !ok {
			if wantsJSON(r) {
				writeError(w, http.StatusUnauthorized, "Admin login required.")
			} else {
				http.Redirect(w, r, "/login", http.StatusFound)
			}
			return
		}
	}
	if isBlogFrontendPath(path) && !s.blogSiteAllowsRequest(r) {
		s.handleBlogClosedPage(w, r)
		return
	}
	if isPrivateCommunityFrontendPath(path) && !s.communityFrontendAllowsRequest(r) {
		s.handleCommunityLoginRequired(w, r)
		return
	}
	if isPrivateCommunityAPIRequest(path, r.Method) && !s.communityFrontendAllowsRequest(r) {
		s.handleCommunityLoginRequired(w, r)
		return
	}
	if isMutating(r.Method) && s.requiresAdminCSRF(path) {
		if err := s.checkCSRF(r); err != nil {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
	}

	s.route(w, r, path)
}

func (s *Server) setSecurityHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
	w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' https://unpkg.com 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob: https:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'self'; base-uri 'self'; form-action 'self'")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Admin-Token, X-User-Token, X-CSRF-Token")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
}
