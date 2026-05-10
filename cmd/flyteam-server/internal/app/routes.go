package app

import (
	"net/http"
	"strings"
)

func (s *Server) route(w http.ResponseWriter, r *http.Request, path string) {
	if s.routeStaticFiles(w, r, path) {
		return
	}
	if s.routeAdminBackendPage(w, r, path) {
		return
	}
	if s.routePublicFrontend(w, r, path) {
		return
	}
	if s.routeUserFrontend(w, r, path) {
		return
	}
	if !strings.HasPrefix(path, "/api/") {
		http.NotFound(w, r)
		return
	}
	s.routeAPI(w, r, path)
}

func (s *Server) routeAPI(w http.ResponseWriter, r *http.Request, path string) {
	if s.routeAdminBackendAPI(w, r, path) {
		return
	}
	if s.requiresSiteAdminAPI(path, r.Method) {
		if _, ok := s.requireSiteAdmin(w, r); !ok {
			return
		}
	}
	if isCommunityAPIPath(path) && !s.blogSiteAllowsRequest(r) {
		writeError(w, http.StatusServiceUnavailable, s.loadBlogSiteState().Notice)
		return
	}
	if s.routePublicAPI(w, r, path) {
		return
	}
	if s.routeSiteAdminContentAPI(w, r, path) {
		return
	}
	if s.routeRecruitAPI(w, r, path) {
		return
	}
	if s.routeSystemAPI(w, r, path) {
		return
	}
	if s.routeCommunityAPI(w, r, path) {
		return
	}
	writeError(w, http.StatusNotFound, "Not found.")
}
