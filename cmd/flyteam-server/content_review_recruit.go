package main

import (
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"strings"
)

func (s *Server) findReviewAlbum(data M, id string) M {
	for _, it := range asList(data["review_albums"]) {
		m := asMap(it)
		if asString(m["id"]) == id {
			return m
		}
	}
	return nil
}
func (s *Server) handleGetReviewAlbum(w http.ResponseWriter, r *http.Request, id string) {
	data := s.loadTeamContent()
	if m := s.findReviewAlbum(data, id); m != nil {
		writeJSON(w, 200, map[string]any{"album": m})
		return
	}
	writeError(w, 404, "Review album not found.")
}
func (s *Server) handleAddReviewAlbum(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req rawReviewAlbumRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "Invalid JSON.")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeError(w, 400, "Title is required.")
		return
	}
	imgs := cleanURLList(req.ImageURLs)
	cover := strings.TrimSpace(req.CoverURL)
	if cover != "" && !contains(imgs, cover) {
		imgs = append([]string{cover}, imgs...)
	}
	if cover == "" && len(imgs) > 0 {
		cover = imgs[0]
	}
	data := s.loadTeamContent()
	item := M{"id": randomHex(5), "title": strings.TrimSpace(req.Title), "date": strings.TrimSpace(req.Date), "category": strings.TrimSpace(req.Category), "summary": strings.TrimSpace(req.Summary), "content": strings.TrimSpace(req.Content), "cover_url": cover, "image_urls": imgs, "pinned": req.Pinned, "created_at": nowISO(), "updated_at": ""}
	data["review_albums"] = append(asList(data["review_albums"]), item)
	s.saveTeamContent(data)
	writeJSON(w, 200, map[string]any{"album": item})
}
func (s *Server) handleUpdateReviewAlbum(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req rawReviewAlbumRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "Invalid JSON.")
		return
	}
	data := s.loadTeamContent()
	m := s.findReviewAlbum(data, id)
	if m == nil {
		writeError(w, 404, "Review album not found.")
		return
	}
	imgs := cleanURLList(req.ImageURLs)
	cover := strings.TrimSpace(req.CoverURL)
	if cover != "" && !contains(imgs, cover) {
		imgs = append([]string{cover}, imgs...)
	}
	if cover == "" && len(imgs) > 0 {
		cover = imgs[0]
	}
	m["title"] = strings.TrimSpace(req.Title)
	m["date"] = strings.TrimSpace(req.Date)
	m["category"] = strings.TrimSpace(req.Category)
	m["summary"] = strings.TrimSpace(req.Summary)
	m["content"] = strings.TrimSpace(req.Content)
	m["cover_url"] = cover
	m["image_urls"] = imgs
	m["pinned"] = req.Pinned
	m["created_at"] = defaultString(asString(m["created_at"]), asString(m["date"]))
	m["updated_at"] = nowISO()
	s.saveTeamContent(data)
	writeJSON(w, 200, map[string]any{"album": m})
}
func (s *Server) handleDeleteReviewAlbumImage(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req GalleryDeleteRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "Invalid JSON.")
		return
	}
	data := s.loadTeamContent()
	m := s.findReviewAlbum(data, id)
	if m == nil {
		writeError(w, 404, "Review album not found.")
		return
	}
	url := strings.TrimSpace(req.URL)
	imgs := stringList(m["image_urls"])
	if !contains(imgs, url) && url != asString(m["cover_url"]) {
		writeError(w, 404, "Image not found in review album.")
		return
	}
	next := []string{}
	for _, u := range imgs {
		if u != url {
			next = append(next, u)
		}
	}
	m["image_urls"] = next
	if asString(m["cover_url"]) == url {
		if len(next) > 0 {
			m["cover_url"] = next[0]
		} else {
			m["cover_url"] = ""
		}
	}
	deleteUploadedImage(url, s.cfg.ReviewUploadDir, "/uploads/review")
	s.saveTeamContent(data)
	writeJSON(w, 200, map[string]any{"album": m, "deleted": url})
}
func (s *Server) handleDeleteReviewAlbum(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	data := s.loadTeamContent()
	out := []any{}
	found := M(nil)
	for _, it := range asList(data["review_albums"]) {
		m := asMap(it)
		if asString(m["id"]) == id {
			found = m
		} else {
			out = append(out, it)
		}
	}
	if found == nil {
		writeError(w, 404, "Review album not found.")
		return
	}
	urls := map[string]bool{}
	for _, u := range stringList(found["image_urls"]) {
		urls[u] = true
	}
	if c := asString(found["cover_url"]); c != "" {
		urls[c] = true
	}
	data["review_albums"] = out
	for u := range urls {
		deleteUploadedImage(u, s.cfg.ReviewUploadDir, "/uploads/review")
	}
	s.saveTeamContent(data)
	writeJSON(w, 200, map[string]any{"deleted": id})
}
func (s *Server) handleAddReview(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req rawReviewRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "Invalid JSON.")
		return
	}
	if strings.TrimSpace(req.ImageURL) == "" {
		writeError(w, 400, "Image URL is required.")
		return
	}
	data := s.loadTeamContent()
	item := M{"id": randomHex(5), "url": strings.TrimSpace(req.ImageURL), "title": strings.TrimSpace(req.Title), "description": strings.TrimSpace(req.Description), "pinned": req.Pinned, "created_at": nowISO(), "updated_at": ""}
	data["review_images"] = append(asList(data["review_images"]), item)
	s.saveTeamContent(data)
	writeJSON(w, 200, map[string]any{"review": item})
}
func (s *Server) handleUpdateReview(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req rawReviewRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "Invalid JSON.")
		return
	}
	data := s.loadTeamContent()
	for _, it := range asList(data["review_images"]) {
		m := asMap(it)
		if asString(m["id"]) == id {
			m["title"] = strings.TrimSpace(req.Title)
			m["description"] = strings.TrimSpace(req.Description)
			m["pinned"] = req.Pinned
			m["updated_at"] = nowISO()
			s.saveTeamContent(data)
			writeJSON(w, 200, map[string]any{"review": m})
			return
		}
	}
	writeError(w, 404, "Review item not found.")
}
func (s *Server) handleDeleteReview(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	data := s.loadTeamContent()
	out := []any{}
	found := M(nil)
	for _, it := range asList(data["review_images"]) {
		m := asMap(it)
		if asString(m["id"]) == id {
			found = m
		} else {
			out = append(out, it)
		}
	}
	if found == nil {
		writeError(w, 404, "Review item not found.")
		return
	}
	data["review_images"] = out
	deleteUploadedImage(asString(found["url"]), s.cfg.ReviewUploadDir, "/uploads/review")
	s.saveTeamContent(data)
	writeJSON(w, 200, map[string]any{"deleted": id})
}

func (s *Server) handleDeleteGallery(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req GalleryDeleteRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "Invalid JSON.")
		return
	}
	data := s.loadTeamContent()
	url := strings.TrimSpace(req.URL)
	out := []any{}
	found := false
	for _, it := range asList(data["gallery"]) {
		if asString(it) == url {
			found = true
		} else {
			out = append(out, it)
		}
	}
	if !found {
		writeError(w, 404, "Image not found in gallery.")
		return
	}
	data["gallery"] = out
	deleteUploadedImage(url, s.cfg.ImageUploadDir, "/uploads/images")
	s.saveTeamContent(data)
	writeJSON(w, 200, map[string]any{"deleted": url})
}
func (s *Server) handleDeleteReviewByURL(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req GalleryDeleteRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "Invalid JSON.")
		return
	}
	data := s.loadTeamContent()
	url := strings.TrimSpace(req.URL)
	out := []any{}
	found := false
	for _, it := range asList(data["review_images"]) {
		m := asMap(it)
		if asString(m["url"]) == url {
			found = true
		} else {
			out = append(out, it)
		}
	}
	if !found {
		writeError(w, 404, "Image not found in review.")
		return
	}
	data["review_images"] = out
	deleteUploadedImage(url, s.cfg.ReviewUploadDir, "/uploads/review")
	s.saveTeamContent(data)
	writeJSON(w, 200, map[string]any{"deleted": url})
}

func (s *Server) loadRecruitContent() []any {
	b, err := os.ReadFile(s.cfg.RecruitContentFile)
	if err != nil {
		return []any{}
	}
	var raw []any
	if json.Unmarshal(b, &raw) != nil {
		return []any{}
	}
	out := []any{}
	for _, it := range raw {
		m := asMap(it)
		if m == nil {
			continue
		}
		out = append(out, M{"id": defaultString(asString(m["id"]), randomHex(6)), "name": asString(m["name"]), "student_id": asString(m["student_id"]), "college": asString(m["college"]), "grade": asString(m["grade"]), "phone": asString(m["phone"]), "wechat": asString(m["wechat"]), "email": asString(m["email"]), "hall": validHall(asString(m["hall"])), "direction_detail": asString(m["direction_detail"]), "experience": asString(m["experience"]), "weekly_hours": asString(m["weekly_hours"]), "note": asString(m["note"]), "pinned": asBool(m["pinned"]), "created_at": asString(m["created_at"]), "updated_at": asString(m["updated_at"])})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return recordLess(asMap(out[i]), asMap(out[j]), []string{"created_at", "date", "year", "grade"})
	})
	return out
}
func (s *Server) saveRecruitContent(items []any) {
	_ = writeJSONAtomic(s.cfg.RecruitContentFile, items)
}
func (s *Server) handleRecruitHalls(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"binary": "二进制（RE / PWN）", "web": "Web（含 Misc / 密码）", "dev": "开发", "management": "团队管理"})
}
func (s *Server) handleRecruitList(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, 200, map[string]any{"items": s.loadRecruitContent()})
}
func (s *Server) handleRecruitStats(w http.ResponseWriter, r *http.Request) {
	items := s.loadRecruitContent()
	stats := map[string]int{"binary": 0, "web": 0, "dev": 0, "management": 0}
	for _, it := range items {
		h := asString(asMap(it)["hall"])
		if _, ok := stats[h]; ok {
			stats[h]++
		}
	}
	writeJSON(w, 200, map[string]any{"stats": stats, "total": len(items)})
}
func (s *Server) handleRecruitApply(w http.ResponseWriter, r *http.Request) {
	var req rawRecruitRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "Invalid JSON.")
		return
	}
	if _, ok := s.adminFromRequest(r); ok {
		if err := s.checkCSRF(r); err != nil {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
	} else if !s.verifyRecruitCaptcha(req.CaptchaToken, req.CaptchaAnswer, clientIP(r)) {
		writeError(w, 400, "验证码错误或已过期，请刷新后重试。")
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.StudentID) == "" {
		writeError(w, 400, "Name and student_id are required.")
		return
	}
	items := s.loadRecruitContent()
	item := M{"id": randomHex(6), "name": strings.TrimSpace(req.Name), "student_id": strings.TrimSpace(req.StudentID), "college": strings.TrimSpace(req.College), "grade": strings.TrimSpace(req.Grade), "phone": strings.TrimSpace(req.Phone), "wechat": strings.TrimSpace(req.Wechat), "email": strings.TrimSpace(req.Email), "hall": validHall(req.Hall), "direction_detail": strings.TrimSpace(req.DirectionDetail), "experience": strings.TrimSpace(req.Experience), "weekly_hours": strings.TrimSpace(req.WeeklyHours), "note": strings.TrimSpace(req.Note), "pinned": false, "created_at": nowISO(), "updated_at": ""}
	items = append(items, item)
	s.saveRecruitContent(items)
	writeJSON(w, 200, map[string]any{"item": item})
}
func (s *Server) handleRecruitUpdate(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req rawRecruitRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "Invalid JSON.")
		return
	}
	items := s.loadRecruitContent()
	for _, it := range items {
		m := asMap(it)
		if asString(m["id"]) == id {
			m["name"] = strings.TrimSpace(req.Name)
			m["student_id"] = strings.TrimSpace(req.StudentID)
			m["college"] = strings.TrimSpace(req.College)
			m["grade"] = strings.TrimSpace(req.Grade)
			m["phone"] = strings.TrimSpace(req.Phone)
			m["wechat"] = strings.TrimSpace(req.Wechat)
			m["email"] = strings.TrimSpace(req.Email)
			m["hall"] = validHall(req.Hall)
			m["direction_detail"] = strings.TrimSpace(req.DirectionDetail)
			m["experience"] = strings.TrimSpace(req.Experience)
			m["weekly_hours"] = strings.TrimSpace(req.WeeklyHours)
			m["note"] = strings.TrimSpace(req.Note)
			m["pinned"] = req.Pinned
			m["created_at"] = asString(m["created_at"])
			m["updated_at"] = nowISO()
			s.saveRecruitContent(items)
			writeJSON(w, 200, map[string]any{"item": m})
			return
		}
	}
	writeError(w, 404, "Recruit application not found.")
}
func (s *Server) handleRecruitDelete(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	items := s.loadRecruitContent()
	out := []any{}
	found := false
	for _, it := range items {
		m := asMap(it)
		if asString(m["id"]) == id {
			found = true
		} else {
			out = append(out, it)
		}
	}
	if !found {
		writeError(w, 404, "Recruit application not found.")
		return
	}
	s.saveRecruitContent(out)
	writeJSON(w, 200, map[string]any{"deleted": id})
}
