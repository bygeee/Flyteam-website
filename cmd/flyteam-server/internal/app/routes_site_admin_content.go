package app

import (
	"net/http"
	"strings"
)

// routeSiteAdminContentAPI contains promotional-site content management APIs.
func (s *Server) routeSiteAdminContentAPI(w http.ResponseWriter, r *http.Request, path string) bool {
	switch {
	case path == "/api/awards" && r.Method == http.MethodPost:
		s.handleAddAward(w, r)
	case strings.HasPrefix(path, "/api/awards/") && r.Method == http.MethodPut:
		s.handleUpdateAward(w, r, pathValue(path, "/api/awards/"))
	case strings.HasPrefix(path, "/api/awards/") && r.Method == http.MethodDelete:
		s.handleDeleteAward(w, r, pathValue(path, "/api/awards/"))
	case path == "/api/seniors" && r.Method == http.MethodPost:
		s.handleAddSenior(w, r)
	case strings.HasPrefix(path, "/api/seniors/") && r.Method == http.MethodPut:
		s.handleUpdateSenior(w, r, pathValue(path, "/api/seniors/"))
	case strings.HasPrefix(path, "/api/seniors/") && r.Method == http.MethodDelete:
		s.handleDeleteSenior(w, r, pathValue(path, "/api/seniors/"))
	case path == "/api/news" && r.Method == http.MethodPost:
		s.handleAddNews(w, r)
	case strings.HasPrefix(path, "/api/news/") && r.Method == http.MethodPut:
		s.handleUpdateNews(w, r, pathValue(path, "/api/news/"))
	case strings.HasPrefix(path, "/api/news/") && r.Method == http.MethodDelete:
		s.handleDeleteNews(w, r, pathValue(path, "/api/news/"))
	case path == "/api/content/intro" && r.Method == http.MethodPost:
		s.handleSaveIntro(w, r)
	case path == "/api/content/overview" && r.Method == http.MethodPost:
		s.handleSaveOverview(w, r)
	case strings.HasPrefix(path, "/api/review/albums/") && strings.HasSuffix(path, "/image/delete") && r.Method == http.MethodPost:
		s.handleDeleteReviewAlbumImage(w, r, strings.TrimSuffix(pathValue(path, "/api/review/albums/"), "/image/delete"))
	case path == "/api/review/albums" && r.Method == http.MethodPost:
		s.handleAddReviewAlbum(w, r)
	case strings.HasPrefix(path, "/api/review/albums/") && r.Method == http.MethodPut:
		s.handleUpdateReviewAlbum(w, r, pathValue(path, "/api/review/albums/"))
	case strings.HasPrefix(path, "/api/review/albums/") && r.Method == http.MethodDelete:
		s.handleDeleteReviewAlbum(w, r, pathValue(path, "/api/review/albums/"))
	case path == "/api/review" && r.Method == http.MethodPost:
		s.handleAddReview(w, r)
	case strings.HasPrefix(path, "/api/review/") && r.Method == http.MethodPut:
		s.handleUpdateReview(w, r, pathValue(path, "/api/review/"))
	case strings.HasPrefix(path, "/api/review/") && r.Method == http.MethodDelete:
		s.handleDeleteReview(w, r, pathValue(path, "/api/review/"))
	case path == "/api/content/gallery/delete" && r.Method == http.MethodPost:
		s.handleDeleteGallery(w, r)
	case path == "/api/content/review/delete" && r.Method == http.MethodPost:
		s.handleDeleteReviewByURL(w, r)
	default:
		return false
	}
	return true
}
