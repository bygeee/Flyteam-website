package main

import (
	"html"
	"net/http"
	"strings"
)

const blogSiteStateKey = "blog_site_state"

type BlogSiteState struct {
	Open      bool   `json:"open"`
	Notice    string `json:"notice"`
	UpdatedAt string `json:"updated_at"`
	UpdatedBy string `json:"updated_by"`
}

type BlogSiteStateUpdateRequest struct {
	Open   *bool  `json:"open"`
	Notice string `json:"notice"`
}

func defaultBlogSiteState() BlogSiteState {
	return BlogSiteState{Open: true, Notice: "\u535a\u5ba2\u7ad9\u6682\u4e0d\u5bf9\u5916\u5f00\u653e\uff0c\u8bf7\u7a0d\u540e\u518d\u8bd5\u3002"}
}

func cleanBlogSiteNotice(raw string) string {
	notice := strings.TrimSpace(raw)
	if notice == "" {
		notice = defaultBlogSiteState().Notice
	}
	runes := []rune(notice)
	if len(runes) > 160 {
		notice = string(runes[:160])
	}
	return notice
}

func (s *Server) loadBlogSiteState() BlogSiteState {
	state := defaultBlogSiteState()
	if s.db == nil {
		return state
	}
	var stored BlogSiteState
	if s.loadJSONFromDB(blogSiteStateKey, &stored) {
		stored.Notice = cleanBlogSiteNotice(stored.Notice)
		return stored
	}
	return state
}

func (s *Server) saveBlogSiteState(state BlogSiteState) error {
	state.Notice = cleanBlogSiteNotice(state.Notice)
	if state.UpdatedAt == "" {
		state.UpdatedAt = nowISO()
	}
	if s.db == nil {
		return nil
	}
	return s.saveJSONToDB(blogSiteStateKey, state)
}

func (s *Server) canBypassBlogSiteClosure(r *http.Request) bool {
	admin, ok := s.adminFromRequest(r)
	return ok && canManageBlogRole(admin.Role)
}

func (s *Server) blogSiteAllowsRequest(r *http.Request) bool {
	state := s.loadBlogSiteState()
	return state.Open || s.canBypassBlogSiteClosure(r)
}

func isBlogFrontendPath(path string) bool {
	if path == "/blog" || strings.HasPrefix(path, "/blog/") || path == "/editor" || path == "/account" || path == "/messages" || path == "/groups" || path == "/user-login" || path == "/user-register" {
		return true
	}
	if strings.HasPrefix(path, "/space/") {
		return true
	}
	switch path {
	case "/static/blog.html", "/static/article.html", "/static/editor.html", "/static/account.html", "/static/messages.html", "/static/groups.html", "/static/space.html", "/static/user_login.html", "/static/user_register.html":
		return true
	default:
		return false
	}
}

func isCommunityAPIPath(path string) bool {
	if path == "/api/search" || path == "/api/upload/blog/images" || path == "/api/upload/avatar" {
		return true
	}
	for _, prefix := range []string{"/api/users", "/api/blog", "/api/social", "/api/friends", "/api/messages", "/api/groups", "/api/notifications"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func (s *Server) handleBlogClosedPage(w http.ResponseWriter, r *http.Request) {
	state := s.loadBlogSiteState()
	notice := html.EscapeString(cleanBlogSiteNotice(state.Notice))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Retry-After", "300")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`<!DOCTYPE html><html lang="zh-CN"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>&#21338;&#23458;&#31449;&#26242;&#19981;&#24320;&#25918;</title><style>body{margin:0;min-height:100vh;display:grid;place-items:center;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:radial-gradient(circle at top,#ffe7ef,#f6f9ff 48%,#eaf1ff);color:#152238}.card{width:min(560px,calc(100% - 40px));padding:38px;border-radius:30px;background:rgba(255,255,255,.78);box-shadow:0 24px 70px rgba(65,84,130,.18);border:1px solid rgba(255,255,255,.75);text-align:center}.badge{display:inline-flex;padding:8px 14px;border-radius:999px;background:#fff0f4;color:#c73355;font-weight:800}h1{font-size:34px;margin:18px 0 12px}p{line-height:1.8;color:#516070;white-space:pre-wrap}.actions{display:flex;justify-content:center;gap:12px;flex-wrap:wrap;margin-top:24px}a{padding:11px 18px;border-radius:999px;text-decoration:none;font-weight:800}.primary{background:#c73355;color:#fff}.ghost{background:#fff;color:#c73355;border:1px solid #ffd1dc}</style></head><body><main class="card"><span class="badge">Flyteam Blog</span><h1>&#21338;&#23458;&#31449;&#26242;&#19981;&#23545;&#22806;&#24320;&#25918;</h1><p>` + notice + `</p><p>&#25968;&#25454;&#12289;&#25991;&#31456;&#12289;&#32842;&#22825;&#35760;&#24405;&#21644;&#19978;&#20256;&#32531;&#23384;&#37117;&#24050;&#20445;&#30041;&#65292;&#21482;&#26159;&#24403;&#21069;&#23545;&#22806;&#35775;&#38382;&#34987;&#20020;&#26102;&#20851;&#38381;&#12290;</p><div class="actions"><a class="primary" href="/">&#36820;&#22238;&#23459;&#20256;&#31449;&#39318;&#39029;</a><a class="ghost" href="/admin">&#31649;&#29702;&#21592;&#21518;&#21488;</a></div></main></body></html>`))
}

func (s *Server) handleGetBlogSiteState(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireBlogAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"state": s.loadBlogSiteState()})
}

func (s *Server) handleUpdateBlogSiteState(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireSuperAdmin(w, r)
	if !ok {
		return
	}
	var req BlogSiteStateUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	if req.Open == nil {
		writeError(w, http.StatusBadRequest, "Open field is required.")
		return
	}
	state := s.loadBlogSiteState()
	state.Open = *req.Open
	state.Notice = cleanBlogSiteNotice(req.Notice)
	state.UpdatedAt = nowISO()
	state.UpdatedBy = admin.Username
	if err := s.saveBlogSiteState(state); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save blog site state.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}
