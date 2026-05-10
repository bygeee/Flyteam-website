package app

import "net/http"

// routeSystemAPI contains RAG, upload, and utility APIs.
func (s *Server) routeSystemAPI(w http.ResponseWriter, r *http.Request, path string) bool {
	switch {
	case path == "/api/ingest/default" && r.Method == http.MethodPost:
		s.handleIngestDefault(w, r)
	case path == "/api/ingest/rebuild/default" && r.Method == http.MethodPost:
		s.handleRebuildDefault(w, r)
	case path == "/api/ingest/local" && r.Method == http.MethodPost:
		s.handleIngestLocal(w, r)
	case path == "/api/upload" && r.Method == http.MethodPost:
		s.handleUploadPDF(w, r)
	case path == "/api/upload/images" && r.Method == http.MethodPost:
		s.handleUploadImages(w, r, s.cfg.ImageUploadDir, "/uploads/images", true)
	case path == "/api/upload/awards/images" && r.Method == http.MethodPost:
		s.handleUploadImages(w, r, s.cfg.AwardUploadDir, "/uploads/awards", false)
	case path == "/api/upload/seniors/images" && r.Method == http.MethodPost:
		s.handleUploadImages(w, r, s.cfg.SeniorUploadDir, "/uploads/seniors", false)
	case path == "/api/upload/review/images" && r.Method == http.MethodPost:
		s.handleUploadImages(w, r, s.cfg.ReviewUploadDir, "/uploads/review", false)
	case path == "/api/upload/news/images" && r.Method == http.MethodPost:
		s.handleUploadImages(w, r, s.cfg.NewsUploadDir, "/uploads/news", false)
	case path == "/api/chat/stream" && r.Method == http.MethodPost:
		s.handleChatStream(w, r)
	case path == "/api/chat" && r.Method == http.MethodPost:
		s.handleChat(w, r)
	default:
		return false
	}
	return true
}
