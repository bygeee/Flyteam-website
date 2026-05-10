package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type M map[string]any

var (
	seniorFullYearRe  = regexp.MustCompile(`20(1[3-9]|2[0-9])|19[0-9]{2}`)
	seniorShortYearRe = regexp.MustCompile(`(^|[^0-9])(1[3-9]|2[0-9])([^0-9]|$)`)
)

type AwardRequest struct {
	Title, AwardType, Year, Level, Organizer, Description, ImageURL string
	Pinned                                                          bool
}
type SeniorRequest struct {
	Name, Grade, Hall, Direction, Intro, Achievements, Advice, PhotoURL string
	Pinned, Responsible                                                 bool
}
type NewsRequest struct {
	Title, Date, Summary, Source, Content, CoverURL string
	ImageURLs                                       []string
	Pinned                                          bool
}
type IntroRequest struct {
	Intro string `json:"intro"`
}
type OverviewRequest struct {
	Overview string `json:"overview"`
}
type ReviewRequest struct {
	ImageURL, Title, Description string
	Pinned                       bool
}
type ReviewAlbumRequest struct {
	Title, Date, Category, Summary, Content, CoverURL string
	ImageURLs                                         []string
	Pinned                                            bool
}
type GalleryDeleteRequest struct {
	URL string `json:"url"`
}
type RecruitRequest struct {
	Name, StudentID, College, Grade, Phone, Wechat, Email, Hall, DirectionDetail, Experience, WeeklyHours, Note, CaptchaToken, CaptchaAnswer string
	Pinned                                                                                                                                   bool
}

type rawAwardRequest struct {
	Title       string `json:"title"`
	AwardType   string `json:"award_type"`
	Year        string `json:"year"`
	Level       string `json:"level"`
	Organizer   string `json:"organizer"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	Pinned      bool   `json:"pinned"`
}
type rawSeniorRequest struct {
	Name         string `json:"name"`
	Grade        string `json:"grade"`
	Hall         string `json:"hall"`
	Direction    string `json:"direction"`
	Intro        string `json:"intro"`
	Achievements string `json:"achievements"`
	Advice       string `json:"advice"`
	PhotoURL     string `json:"photo_url"`
	Pinned       bool   `json:"pinned"`
	Responsible  bool   `json:"responsible"`
}
type rawNewsRequest struct {
	Title     string   `json:"title"`
	Date      string   `json:"date"`
	Summary   string   `json:"summary"`
	Source    string   `json:"source"`
	Content   string   `json:"content"`
	CoverURL  string   `json:"cover_url"`
	ImageURLs []string `json:"image_urls"`
	Pinned    bool     `json:"pinned"`
}
type rawReviewRequest struct {
	ImageURL    string `json:"image_url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Pinned      bool   `json:"pinned"`
}
type rawReviewAlbumRequest struct {
	Title     string   `json:"title"`
	Date      string   `json:"date"`
	Category  string   `json:"category"`
	Summary   string   `json:"summary"`
	Content   string   `json:"content"`
	CoverURL  string   `json:"cover_url"`
	ImageURLs []string `json:"image_urls"`
	Pinned    bool     `json:"pinned"`
}
type rawRecruitRequest struct {
	Name            string `json:"name"`
	StudentID       string `json:"student_id"`
	College         string `json:"college"`
	Grade           string `json:"grade"`
	Phone           string `json:"phone"`
	Wechat          string `json:"wechat"`
	Email           string `json:"email"`
	Hall            string `json:"hall"`
	DirectionDetail string `json:"direction_detail"`
	Experience      string `json:"experience"`
	WeeklyHours     string `json:"weekly_hours"`
	Note            string `json:"note"`
	CaptchaToken    string `json:"captcha_token"`
	CaptchaAnswer   string `json:"captcha_answer"`
	Pinned          bool   `json:"pinned"`
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case float64:
		return strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(strconvFormat(x), "0"), "."))
	default:
		return strings.TrimSpace(toJSONScalar(x))
	}
}
func strconvFormat(f float64) string {
	return strings.TrimRight(strings.TrimRight(fmtFloat(f), "0"), ".")
}
func fmtFloat(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }
func toJSONScalar(v any) string { b, _ := json.Marshal(v); return strings.Trim(string(b), "\"") }
func asBool(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case float64:
		return x != 0
	case string:
		y := strings.ToLower(strings.TrimSpace(x))
		return y == "true" || y == "1" || y == "yes" || y == "on" || y == "负责人"
	}
	return false
}
func asMap(v any) M {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	if m, ok := v.(M); ok {
		return m
	}
	return nil
}
func asList(v any) []any {
	if a, ok := v.([]any); ok {
		return a
	}
	return []any{}
}
func stringList(v any) []string {
	out := []string{}
	switch a := v.(type) {
	case []any:
		for _, x := range a {
			if s := asString(x); s != "" {
				out = append(out, s)
			}
		}
	case []string:
		for _, s := range a {
			if strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
	}
	return cleanURLList(out)
}

func cleanURLList(urls []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u != "" && !seen[u] {
			seen[u] = true
			out = append(out, u)
		}
	}
	return out
}

func normalizeAwardType(v any) string {
	raw := strings.ToLower(asString(v))
	if raw == "personal" || raw == "individual" || raw == "solo" || strings.Contains(raw, "个人") {
		return "personal"
	}
	return "team"
}
func normalizeAwardLevel(v any) string {
	raw := asString(v)
	low := strings.ToLower(raw)
	if low == "national" || strings.Contains(raw, "国家") || strings.Contains(raw, "全国") {
		return "国家级"
	}
	return "省级"
}
func awardLevelRank(item M) int {
	if normalizeAwardLevel(item["level"]) == "国家级" {
		return 1
	}
	return 0
}

func (s *Server) loadTeamContent() M {
	empty := M{"awards": []any{}, "gallery": []any{}, "seniors": []any{}, "news": []any{}, "review_images": []any{}, "review_albums": []any{}, "team_intro": "", "team_overview": ""}
	var data M
	fromDB := s.loadJSONFromDB("team_content", &data) && data != nil
	if !fromDB {
		b, err := os.ReadFile(s.cfg.TeamContentFile)
		if err != nil {
			return empty
		}
		if json.Unmarshal(b, &data) != nil || data == nil {
			return empty
		}
	}
	for k, v := range empty {
		if _, ok := data[k]; !ok {
			data[k] = v
		}
	}

	awards := []any{}
	for _, it := range asList(data["awards"]) {
		m := asMap(it)
		if m == nil {
			continue
		}
		title := asString(m["title"])
		img := asString(m["image_url"])
		if title == "" && img == "" {
			continue
		}
		awards = append(awards, M{"id": defaultString(asString(m["id"]), randomHex(5)), "title": title, "award_type": normalizeAwardType(firstNonNil(m["award_type"], m["category"], m["type"])), "year": asString(m["year"]), "level": normalizeAwardLevel(m["level"]), "organizer": asString(m["organizer"]), "description": asString(m["description"]), "image_url": img, "pinned": asBool(m["pinned"]), "created_at": defaultString(asString(m["created_at"]), asString(m["year"])), "updated_at": asString(m["updated_at"])})
	}
	sort.SliceStable(awards, func(i, j int) bool { return awardLess(asMap(awards[i]), asMap(awards[j])) })
	data["awards"] = awards

	seniors := []any{}
	for _, it := range asList(data["seniors"]) {
		m := asMap(it)
		if m == nil {
			continue
		}
		name := asString(m["name"])
		photo := asString(m["photo_url"])
		if name == "" && photo == "" {
			continue
		}
		seniors = append(seniors, M{"id": defaultString(asString(m["id"]), randomHex(5)), "name": name, "grade": asString(m["grade"]), "hall": validHall(asString(m["hall"])), "direction": asString(m["direction"]), "intro": asString(m["intro"]), "achievements": asString(m["achievements"]), "advice": asString(m["advice"]), "photo_url": photo, "pinned": asBool(m["pinned"]), "responsible": asBool(firstNonNil(m["responsible"], m["is_responsible"], m["is_manager"])), "created_at": defaultString(asString(m["created_at"]), asString(m["grade"])), "updated_at": asString(m["updated_at"])})
	}
	sort.SliceStable(seniors, func(i, j int) bool {
		return seniorLess(asMap(seniors[i]), asMap(seniors[j]))
	})
	data["seniors"] = seniors

	reviews := []any{}
	for _, it := range asList(data["review_images"]) {
		if str, ok := it.(string); ok {
			u := strings.TrimSpace(str)
			if u != "" {
				reviews = append(reviews, M{"id": u, "url": u, "title": strings.TrimSuffix(filepath.Base(u), filepath.Ext(u)), "description": "", "pinned": false, "created_at": "", "updated_at": ""})
			}
			continue
		}
		m := asMap(it)
		if m == nil {
			continue
		}
		u := asString(m["url"])
		if u == "" {
			continue
		}
		reviews = append(reviews, M{"id": defaultString(asString(m["id"]), u), "url": u, "title": asString(m["title"]), "description": asString(m["description"]), "pinned": asBool(m["pinned"]), "created_at": asString(m["created_at"]), "updated_at": asString(m["updated_at"])})
	}
	sort.SliceStable(reviews, func(i, j int) bool {
		return recordLess(asMap(reviews[i]), asMap(reviews[j]), []string{"created_at", "date", "year", "grade"})
	})
	data["review_images"] = reviews

	albums := []any{}
	for _, it := range asList(data["review_albums"]) {
		m := asMap(it)
		if m == nil {
			continue
		}
		imgs := stringList(m["image_urls"])
		single := asString(m["url"])
		if single != "" && !contains(imgs, single) {
			imgs = append([]string{single}, imgs...)
		}
		title := asString(m["title"])
		if title == "" && len(imgs) == 0 {
			continue
		}
		cover := asString(m["cover_url"])
		if cover != "" && !contains(imgs, cover) {
			imgs = append([]string{cover}, imgs...)
		}
		if cover == "" && len(imgs) > 0 {
			cover = imgs[0]
		}
		albums = append(albums, M{"id": defaultString(asString(m["id"]), randomHex(5)), "title": defaultString(title, "团队回顾"), "date": asString(m["date"]), "category": asString(m["category"]), "summary": defaultString(asString(m["summary"]), asString(m["description"])), "content": asString(m["content"]), "cover_url": cover, "image_urls": imgs, "pinned": asBool(m["pinned"]), "created_at": defaultString(asString(m["created_at"]), asString(m["date"])), "updated_at": asString(m["updated_at"])})
	}
	if len(albums) == 0 && len(reviews) > 0 {
		urls := []string{}
		for _, it := range reviews {
			if u := asString(asMap(it)["url"]); u != "" {
				urls = append(urls, u)
			}
		}
		if len(urls) > 0 {
			albums = append(albums, M{"id": "legacy-review", "title": "团队回顾照片", "date": "", "category": "历史回顾", "summary": "历史团队回顾照片合集。", "content": "", "cover_url": urls[0], "image_urls": urls, "pinned": false, "created_at": "", "updated_at": ""})
		}
	}
	sort.SliceStable(albums, func(i, j int) bool {
		return recordLess(asMap(albums[i]), asMap(albums[j]), []string{"created_at", "date", "year", "grade"})
	})
	data["review_albums"] = albums

	news := []any{}
	for _, it := range asList(data["news"]) {
		m := asMap(it)
		if m == nil {
			continue
		}
		title := asString(m["title"])
		if title == "" {
			continue
		}
		news = append(news, M{"id": defaultString(asString(m["id"]), randomHex(5)), "title": title, "date": asString(m["date"]), "summary": asString(m["summary"]), "source": asString(m["source"]), "content": asString(m["content"]), "cover_url": asString(m["cover_url"]), "image_urls": stringList(m["image_urls"]), "pinned": asBool(m["pinned"]), "created_at": defaultString(asString(m["created_at"]), asString(m["date"])), "updated_at": asString(m["updated_at"])})
	}
	sort.SliceStable(news, func(i, j int) bool { return recordLess(asMap(news[i]), asMap(news[j]), []string{"date", "created_at"}) })
	data["news"] = news
	if !fromDB && s.db != nil {
		_ = s.saveJSONToDB("team_content", data)
	}
	return data
}

func (s *Server) saveTeamContent(data M) {
	if s.db != nil {
		_ = s.saveJSONToDB("team_content", data)
		return
	}
	_ = writeJSONAtomic(s.cfg.TeamContentFile, data)
}

func firstNonNil(vals ...any) any {
	for _, v := range vals {
		if v != nil {
			return v
		}
	}
	return nil
}
func defaultString(v, def string) string {
	if strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return def
}
func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func parseSortTimestamp(value string) float64 {
	text := strings.TrimSpace(value)
	if text == "" {
		return 0
	}
	norm := strings.NewReplacer("年", "-", "月", "-", "日", "", "/", "-", ".", "-").Replace(text)
	layouts := []string{"2006-1-2", time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05"}
	for _, lay := range layouts {
		if t, err := time.Parse(lay, norm); err == nil {
			return float64(t.Unix())
		}
	}
	if len(text) >= 4 {
		for i := 0; i+4 <= len(text); i++ {
			y := text[i : i+4]
			if y >= "1900" && y <= "2099" {
				if t, err := time.Parse("2006", "2006"); err == nil {
					_ = t
				}
				yy, _ := strconv.Atoi(y)
				return float64(time.Date(yy, 1, 1, 0, 0, 0, 0, time.UTC).Unix())
			}
		}
	}
	return 0
}
func recordText(m M, fields []string) string {
	for _, f := range fields {
		if v := asString(m[f]); v != "" {
			return v
		}
	}
	return ""
}
func recordValue(m M, fields []string) float64 {
	for _, f := range fields {
		if ts := parseSortTimestamp(asString(m[f])); ts != 0 {
			return ts
		}
	}
	return 0
}
func recordLess(a, b M, fields []string) bool {
	pa, pb := asBool(a["pinned"]), asBool(b["pinned"])
	if pa != pb {
		return pa
	}
	va, vb := recordValue(a, fields), recordValue(b, fields)
	if va != vb {
		return va > vb
	}
	return recordText(a, fields) > recordText(b, fields)
}
func seniorGradeRank(m M) int {
	text := strings.TrimSpace(asString(m["grade"]))
	if text == "" {
		return -1
	}
	low := strings.ToLower(text)
	if strings.Contains(text, "帮主") || strings.Contains(low, "leader") {
		return 100000
	}
	if y := seniorFullYearRe.FindString(text); y != "" {
		if n, err := strconv.Atoi(y); err == nil {
			return n
		}
	}
	if m := seniorShortYearRe.FindStringSubmatch(text); len(m) >= 3 {
		if n, err := strconv.Atoi(m[2]); err == nil {
			return 2000 + n
		}
	}
	return -1
}
func seniorLess(a, b M) bool {
	pa, pb := asBool(a["pinned"]), asBool(b["pinned"])
	if pa != pb {
		return pa
	}
	ga, gb := seniorGradeRank(a), seniorGradeRank(b)
	if ga != gb {
		return ga > gb
	}
	return recordLess(a, b, []string{"created_at", "date", "year", "grade"})
}
func awardLess(a, b M) bool {
	pa, pb := asBool(a["pinned"]), asBool(b["pinned"])
	if pa != pb {
		return pa
	}
	la, lb := awardLevelRank(a), awardLevelRank(b)
	if la != lb {
		return la > lb
	}
	fields := []string{"date", "year", "created_at"}
	va, vb := recordValue(a, fields), recordValue(b, fields)
	if va != vb {
		return va > vb
	}
	return recordText(a, fields) > recordText(b, fields)
}

func (s *Server) handleGetContent(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.loadTeamContent())
}
func (s *Server) handleGetNews(w http.ResponseWriter, r *http.Request, id string) {
	data := s.loadTeamContent()
	for _, it := range asList(data["news"]) {
		m := asMap(it)
		if asString(m["id"]) == id {
			writeJSON(w, 200, map[string]any{"news": m})
			return
		}
	}
	writeError(w, 404, "News not found.")
}

func (s *Server) handleAddAward(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req rawAwardRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "Invalid JSON.")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeError(w, 400, "Title is required.")
		return
	}
	data := s.loadTeamContent()
	item := M{"id": randomHex(5), "title": strings.TrimSpace(req.Title), "award_type": normalizeAwardType(req.AwardType), "year": strings.TrimSpace(req.Year), "level": normalizeAwardLevel(req.Level), "organizer": strings.TrimSpace(req.Organizer), "description": strings.TrimSpace(req.Description), "image_url": strings.TrimSpace(req.ImageURL), "pinned": req.Pinned, "created_at": nowISO(), "updated_at": ""}
	data["awards"] = append(asList(data["awards"]), item)
	s.saveTeamContent(data)
	writeJSON(w, 200, map[string]any{"award": item})
}
func (s *Server) handleUpdateAward(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req rawAwardRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "Invalid JSON.")
		return
	}
	data := s.loadTeamContent()
	arr := asList(data["awards"])
	for _, it := range arr {
		m := asMap(it)
		if asString(m["id"]) == id {
			old := asString(m["image_url"])
			m["title"] = strings.TrimSpace(req.Title)
			m["award_type"] = normalizeAwardType(req.AwardType)
			m["year"] = strings.TrimSpace(req.Year)
			m["level"] = normalizeAwardLevel(req.Level)
			m["organizer"] = strings.TrimSpace(req.Organizer)
			m["description"] = strings.TrimSpace(req.Description)
			m["image_url"] = strings.TrimSpace(req.ImageURL)
			m["pinned"] = req.Pinned
			m["created_at"] = defaultString(asString(m["created_at"]), asString(m["year"]))
			m["updated_at"] = nowISO()
			if old != "" && old != asString(m["image_url"]) {
				deleteUploadedImage(old, s.cfg.AwardUploadDir, "/uploads/awards")
			}
			s.saveTeamContent(data)
			writeJSON(w, 200, map[string]any{"award": m})
			return
		}
	}
	writeError(w, 404, "Award not found.")
}
func (s *Server) handleDeleteAward(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	data := s.loadTeamContent()
	out := []any{}
	found := M(nil)
	for _, it := range asList(data["awards"]) {
		m := asMap(it)
		if asString(m["id"]) == id {
			found = m
		} else {
			out = append(out, it)
		}
	}
	if found == nil {
		writeError(w, 404, "Award not found.")
		return
	}
	data["awards"] = out
	deleteUploadedImage(asString(found["image_url"]), s.cfg.AwardUploadDir, "/uploads/awards")
	s.saveTeamContent(data)
	writeJSON(w, 200, map[string]any{"deleted": id})
}

func (s *Server) handleAddSenior(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req rawSeniorRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "Invalid JSON.")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, 400, "Name is required.")
		return
	}
	data := s.loadTeamContent()
	item := M{"id": randomHex(5), "name": strings.TrimSpace(req.Name), "grade": strings.TrimSpace(req.Grade), "hall": validHall(req.Hall), "direction": strings.TrimSpace(req.Direction), "intro": strings.TrimSpace(req.Intro), "achievements": strings.TrimSpace(req.Achievements), "advice": strings.TrimSpace(req.Advice), "photo_url": strings.TrimSpace(req.PhotoURL), "pinned": req.Pinned, "responsible": req.Responsible, "created_at": nowISO(), "updated_at": ""}
	data["seniors"] = append(asList(data["seniors"]), item)
	s.saveTeamContent(data)
	writeJSON(w, 200, map[string]any{"senior": item})
}
func (s *Server) handleUpdateSenior(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req rawSeniorRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "Invalid JSON.")
		return
	}
	data := s.loadTeamContent()
	for _, it := range asList(data["seniors"]) {
		m := asMap(it)
		if asString(m["id"]) == id {
			old := asString(m["photo_url"])
			m["name"] = strings.TrimSpace(req.Name)
			m["grade"] = strings.TrimSpace(req.Grade)
			m["hall"] = validHall(req.Hall)
			m["direction"] = strings.TrimSpace(req.Direction)
			m["intro"] = strings.TrimSpace(req.Intro)
			m["achievements"] = strings.TrimSpace(req.Achievements)
			m["advice"] = strings.TrimSpace(req.Advice)
			m["photo_url"] = strings.TrimSpace(req.PhotoURL)
			m["pinned"] = req.Pinned
			m["responsible"] = req.Responsible
			m["created_at"] = defaultString(asString(m["created_at"]), asString(m["grade"]))
			m["updated_at"] = nowISO()
			if old != "" && old != asString(m["photo_url"]) {
				deleteUploadedImage(old, s.cfg.SeniorUploadDir, "/uploads/seniors")
			}
			s.saveTeamContent(data)
			writeJSON(w, 200, map[string]any{"senior": m})
			return
		}
	}
	writeError(w, 404, "Senior not found.")
}
func (s *Server) handleDeleteSenior(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	data := s.loadTeamContent()
	out := []any{}
	found := M(nil)
	for _, it := range asList(data["seniors"]) {
		m := asMap(it)
		if asString(m["id"]) == id {
			found = m
		} else {
			out = append(out, it)
		}
	}
	if found == nil {
		writeError(w, 404, "Senior not found.")
		return
	}
	data["seniors"] = out
	deleteUploadedImage(asString(found["photo_url"]), s.cfg.SeniorUploadDir, "/uploads/seniors")
	s.saveTeamContent(data)
	writeJSON(w, 200, map[string]any{"deleted": id})
}

func (s *Server) handleAddNews(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req rawNewsRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "Invalid JSON.")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeError(w, 400, "Title is required.")
		return
	}
	data := s.loadTeamContent()
	item := M{"id": randomHex(5), "title": strings.TrimSpace(req.Title), "date": strings.TrimSpace(req.Date), "summary": strings.TrimSpace(req.Summary), "source": strings.TrimSpace(req.Source), "content": strings.TrimSpace(req.Content), "cover_url": strings.TrimSpace(req.CoverURL), "image_urls": cleanURLList(req.ImageURLs), "pinned": req.Pinned, "created_at": nowISO(), "updated_at": ""}
	data["news"] = append(asList(data["news"]), item)
	s.saveTeamContent(data)
	writeJSON(w, 200, map[string]any{"news": item})
}
func (s *Server) handleUpdateNews(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req rawNewsRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "Invalid JSON.")
		return
	}
	data := s.loadTeamContent()
	for _, it := range asList(data["news"]) {
		m := asMap(it)
		if asString(m["id"]) == id {
			m["title"] = strings.TrimSpace(req.Title)
			m["date"] = strings.TrimSpace(req.Date)
			m["summary"] = strings.TrimSpace(req.Summary)
			m["source"] = strings.TrimSpace(req.Source)
			m["content"] = strings.TrimSpace(req.Content)
			m["cover_url"] = strings.TrimSpace(req.CoverURL)
			m["image_urls"] = cleanURLList(req.ImageURLs)
			m["pinned"] = req.Pinned
			m["created_at"] = defaultString(asString(m["created_at"]), asString(m["date"]))
			m["updated_at"] = nowISO()
			s.saveTeamContent(data)
			writeJSON(w, 200, map[string]any{"news": m})
			return
		}
	}
	writeError(w, 404, "News not found.")
}
func (s *Server) handleDeleteNews(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	data := s.loadTeamContent()
	out := []any{}
	found := M(nil)
	for _, it := range asList(data["news"]) {
		m := asMap(it)
		if asString(m["id"]) == id {
			found = m
		} else {
			out = append(out, it)
		}
	}
	if found == nil {
		writeError(w, 404, "News not found.")
		return
	}
	data["news"] = out
	deleteUploadedImage(asString(found["cover_url"]), s.cfg.NewsUploadDir, "/uploads/news")
	for _, u := range stringList(found["image_urls"]) {
		deleteUploadedImage(u, s.cfg.NewsUploadDir, "/uploads/news")
	}
	s.saveTeamContent(data)
	writeJSON(w, 200, map[string]any{"deleted": id})
}

func (s *Server) handleSaveIntro(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req IntroRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "Invalid JSON.")
		return
	}
	data := s.loadTeamContent()
	data["team_intro"] = strings.TrimSpace(req.Intro)
	s.saveTeamContent(data)
	writeJSON(w, 200, map[string]any{"saved": true})
}
func (s *Server) handleSaveOverview(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req OverviewRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "Invalid JSON.")
		return
	}
	data := s.loadTeamContent()
	data["team_overview"] = strings.TrimSpace(req.Overview)
	s.saveTeamContent(data)
	writeJSON(w, 200, map[string]any{"saved": true})
}
