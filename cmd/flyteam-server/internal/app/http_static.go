package app

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) serveStaticHTML(w http.ResponseWriter, r *http.Request, name string) {
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, filepath.Join(s.cfg.StaticDir, "pages", name))
}

func isAdminStaticAsset(path string) bool {
	switch path {
	case "/static/admin.html", "/static/pages/admin.html", "/static/app.js", "/static/js/app.js":
		return true
	default:
		return false
	}
}

func (s *Server) serveFileRoot(w http.ResponseWriter, r *http.Request, root, rel string) {
	rel = filepath.Clean(strings.TrimPrefix(rel, "/"))
	if rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) || blockedPublicFile(rel) {
		http.NotFound(w, r)
		return
	}
	full := filepath.Join(root, rel)
	if !pathInside(root, full) {
		http.NotFound(w, r)
		return
	}
	if st, err := os.Stat(full); err != nil || st.IsDir() {
		http.NotFound(w, r)
		return
	}
	if ct := mime.TypeByExtension(filepath.Ext(full)); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	http.ServeFile(w, r, full)
}

func blockedPublicFile(rel string) bool {
	normalized := filepath.ToSlash(filepath.Clean(rel))
	for _, part := range strings.Split(normalized, "/") {
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	lower := strings.ToLower(normalized)
	if strings.HasSuffix(lower, "~") {
		return true
	}
	for _, suffix := range []string{
		".codex_backup", ".bak", ".backup", ".old", ".orig", ".tmp", ".temp", ".swp",
		".env", ".log", ".db", ".sqlite", ".sqlite3", ".go", ".py", ".ps1", ".sh", ".bat", ".cmd",
	} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

func pathInside(root, target string) bool {
	rootAbs, err1 := filepath.Abs(root)
	targetAbs, err2 := filepath.Abs(target)
	if err1 != nil || err2 != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != "" && !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel))
}
