package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

type BlogArticle struct {
	ID              string   `json:"id"`
	AuthorID        string   `json:"author_id"`
	AuthorUserID    string   `json:"author_user_id"`
	AuthorNickname  string   `json:"author_nickname"`
	Title           string   `json:"title"`
	Summary         string   `json:"summary"`
	CoverURL        string   `json:"cover_url"`
	ContentMarkdown string   `json:"content_markdown"`
	Tags            []string `json:"tags"`
	Category        string   `json:"category"`
	Status          string   `json:"status"`
	Views           int      `json:"views"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
	PublishedAt     string   `json:"published_at"`
}

type BlogArticleStore struct {
	Articles []BlogArticle `json:"articles"`
}

type BlogArticleRequest struct {
	Title           string   `json:"title"`
	Summary         string   `json:"summary"`
	CoverURL        string   `json:"cover_url"`
	ContentMarkdown string   `json:"content_markdown"`
	Tags            []string `json:"tags"`
	Category        string   `json:"category"`
	Status          string   `json:"status"`
}

func publicBlogArticle(a BlogArticle, includeContent bool) M {
	out := M{
		"id":              a.ID,
		"author_id":       a.AuthorID,
		"author_user_id":  a.AuthorUserID,
		"author_nickname": a.AuthorNickname,
		"title":           a.Title,
		"summary":         a.Summary,
		"cover_url":       a.CoverURL,
		"tags":            a.Tags,
		"category":        a.Category,
		"status":          a.Status,
		"views":           a.Views,
		"created_at":      a.CreatedAt,
		"updated_at":      a.UpdatedAt,
		"published_at":    a.PublishedAt,
	}
	if includeContent {
		out["content_markdown"] = a.ContentMarkdown
	}
	return out
}

func normalizeBlogStatus(status string) string {
	if strings.EqualFold(strings.TrimSpace(status), "published") {
		return "published"
	}
	return "draft"
}

func normalizeBlogTags(tags []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, tag := range tags {
		clean := strings.TrimSpace(tag)
		if clean == "" || seen[strings.ToLower(clean)] {
			continue
		}
		seen[strings.ToLower(clean)] = true
		out = append(out, clean)
		if len(out) >= 8 {
			break
		}
	}
	return out
}

func (s *Server) loadBlogArticles() BlogArticleStore {
	if s.db != nil {
		return s.loadBlogArticlesDB()
	}
	return s.loadBlogArticlesFromJSON()
}

func (s *Server) loadBlogArticlesFromJSON() BlogArticleStore {
	b, err := os.ReadFile(s.cfg.BlogArticlesFile)
	if err != nil {
		return BlogArticleStore{Articles: []BlogArticle{}}
	}
	var store BlogArticleStore
	if json.Unmarshal(b, &store) != nil {
		return BlogArticleStore{Articles: []BlogArticle{}}
	}
	out := BlogArticleStore{Articles: []BlogArticle{}}
	seen := map[string]bool{}
	for _, a := range store.Articles {
		a.ID = strings.TrimSpace(a.ID)
		a.AuthorID = strings.TrimSpace(a.AuthorID)
		a.AuthorUserID = strings.TrimSpace(a.AuthorUserID)
		a.AuthorNickname = strings.TrimSpace(a.AuthorNickname)
		a.Title = strings.TrimSpace(a.Title)
		if a.ID == "" || a.AuthorID == "" || a.Title == "" || seen[a.ID] {
			continue
		}
		seen[a.ID] = true
		a.Status = normalizeBlogStatus(a.Status)
		a.Summary = strings.TrimSpace(a.Summary)
		a.CoverURL = strings.TrimSpace(a.CoverURL)
		a.Category = strings.TrimSpace(a.Category)
		a.Tags = normalizeBlogTags(a.Tags)
		if a.CreatedAt == "" {
			a.CreatedAt = nowISO()
		}
		out.Articles = append(out.Articles, a)
	}
	return out
}

func (s *Server) saveBlogArticles(store BlogArticleStore) error {
	if s.db != nil {
		return s.saveBlogArticlesDB(store)
	}
	for i := range store.Articles {
		store.Articles[i].Title = strings.TrimSpace(store.Articles[i].Title)
		store.Articles[i].Summary = strings.TrimSpace(store.Articles[i].Summary)
		store.Articles[i].CoverURL = strings.TrimSpace(store.Articles[i].CoverURL)
		store.Articles[i].Category = strings.TrimSpace(store.Articles[i].Category)
		store.Articles[i].Status = normalizeBlogStatus(store.Articles[i].Status)
		store.Articles[i].Tags = normalizeBlogTags(store.Articles[i].Tags)
	}
	return writeJSONAtomic(s.cfg.BlogArticlesFile, store)
}

func (s *Server) loadBlogArticlesDB() BlogArticleStore {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM blog_articles`).Scan(&count)
	if count == 0 {
		legacy := s.loadBlogArticlesFromJSON()
		if len(legacy.Articles) > 0 {
			_ = s.saveBlogArticlesDB(legacy)
		}
	}
	rows, err := s.db.Query(`SELECT a.id, a.author_id, COALESCE(u.user_id,''), COALESCE(u.nickname,''), a.title, COALESCE(a.summary,''), COALESCE(a.cover_url,''), a.content_markdown, COALESCE(a.category,''), a.status, a.views, a.created_at, COALESCE(a.updated_at,''), COALESCE(a.published_at,'')
		FROM blog_articles a
		LEFT JOIN community_users u ON u.id = a.author_id
		ORDER BY a.created_at ASC, a.id ASC`)
	if err != nil {
		return BlogArticleStore{Articles: []BlogArticle{}}
	}
	defer rows.Close()
	out := BlogArticleStore{Articles: []BlogArticle{}}
	for rows.Next() {
		var a BlogArticle
		if err := rows.Scan(&a.ID, &a.AuthorID, &a.AuthorUserID, &a.AuthorNickname, &a.Title, &a.Summary, &a.CoverURL, &a.ContentMarkdown, &a.Category, &a.Status, &a.Views, &a.CreatedAt, &a.UpdatedAt, &a.PublishedAt); err != nil {
			return BlogArticleStore{Articles: []BlogArticle{}}
		}
		a.Status = normalizeBlogStatus(a.Status)
		out.Articles = append(out.Articles, a)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return BlogArticleStore{Articles: []BlogArticle{}}
	}
	for i := range out.Articles {
		out.Articles[i].Tags = s.loadBlogArticleTagsDB(out.Articles[i].ID)
	}
	return out
}

func (s *Server) loadBlogArticleTagsDB(articleID string) []string {
	rows, err := s.db.Query(`SELECT tag FROM blog_article_tags WHERE article_id = ? ORDER BY tag ASC`, articleID)
	if err != nil {
		return []string{}
	}
	defer rows.Close()
	tags := []string{}
	for rows.Next() {
		var tag string
		if rows.Scan(&tag) == nil {
			tags = append(tags, tag)
		}
	}
	return normalizeBlogTags(tags)
}

func (s *Server) saveBlogArticlesDB(store BlogArticleStore) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	currentRows, err := tx.Query(`SELECT id FROM blog_articles`)
	if err != nil {
		return err
	}
	existingIDs := []string{}
	for currentRows.Next() {
		var id string
		if currentRows.Scan(&id) == nil {
			existingIDs = append(existingIDs, id)
		}
	}
	currentRows.Close()

	keep := map[string]bool{}
	articleStmt, err := tx.Prepare(`INSERT INTO blog_articles(id, author_id, title, summary, cover_url, content_markdown, status, category, views, published_at, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			author_id=excluded.author_id,
			title=excluded.title,
			summary=excluded.summary,
			cover_url=excluded.cover_url,
			content_markdown=excluded.content_markdown,
			status=excluded.status,
			category=excluded.category,
			views=excluded.views,
			published_at=excluded.published_at,
			created_at=excluded.created_at,
			updated_at=excluded.updated_at`)
	if err != nil {
		return err
	}
	defer articleStmt.Close()

	tagDeleteStmt, err := tx.Prepare(`DELETE FROM blog_article_tags WHERE article_id = ?`)
	if err != nil {
		return err
	}
	defer tagDeleteStmt.Close()

	tagInsertStmt, err := tx.Prepare(`INSERT INTO blog_article_tags(article_id, tag) VALUES(?, ?)`)
	if err != nil {
		return err
	}
	defer tagInsertStmt.Close()

	for i := range store.Articles {
		a := &store.Articles[i]
		a.ID = strings.TrimSpace(a.ID)
		a.AuthorID = strings.TrimSpace(a.AuthorID)
		a.Title = strings.TrimSpace(a.Title)
		if a.ID == "" || a.AuthorID == "" || a.Title == "" {
			continue
		}
		a.Summary = strings.TrimSpace(a.Summary)
		a.CoverURL = strings.TrimSpace(a.CoverURL)
		a.Category = strings.TrimSpace(a.Category)
		a.Status = normalizeBlogStatus(a.Status)
		a.Tags = normalizeBlogTags(a.Tags)
		if a.CreatedAt == "" {
			a.CreatedAt = nowISO()
		}
		keep[a.ID] = true
		publishedAt := sql.NullString{String: strings.TrimSpace(a.PublishedAt), Valid: strings.TrimSpace(a.PublishedAt) != ""}
		updatedAt := sql.NullString{String: strings.TrimSpace(a.UpdatedAt), Valid: strings.TrimSpace(a.UpdatedAt) != ""}
		if _, err := articleStmt.Exec(a.ID, a.AuthorID, a.Title, a.Summary, a.CoverURL, a.ContentMarkdown, a.Status, a.Category, a.Views, publishedAt, a.CreatedAt, updatedAt); err != nil {
			return err
		}
		if _, err := tagDeleteStmt.Exec(a.ID); err != nil {
			return err
		}
		for _, tag := range a.Tags {
			if _, err := tagInsertStmt.Exec(a.ID, tag); err != nil {
				return err
			}
		}
	}

	for _, id := range existingIDs {
		if keep[id] {
			continue
		}
		if _, err := tx.Exec(`DELETE FROM blog_articles WHERE id = ?`, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func findBlogArticle(store BlogArticleStore, id string) (int, *BlogArticle) {
	for i := range store.Articles {
		if store.Articles[i].ID == id {
			return i, &store.Articles[i]
		}
	}
	return -1, nil
}

func canEditBlogArticle(user CommunitySession, article BlogArticle) bool {
	return user.ID != "" && user.ID == article.AuthorID
}

func (s *Server) handleBlogArticles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListBlogArticles(w, r)
	case http.MethodPost:
		s.handleCreateBlogArticle(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
	}
}

func (s *Server) handleListBlogArticles(w http.ResponseWriter, r *http.Request) {
	store := s.loadBlogArticles()
	current, loggedIn := s.communityUserFromRequest(r)
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	mine := r.URL.Query().Get("mine") == "1" || strings.EqualFold(r.URL.Query().Get("mine"), "true")
	items := []M{}
	for _, a := range store.Articles {
		if mine {
			if !loggedIn || a.AuthorID != current.ID {
				continue
			}
		} else if a.Status != "published" {
			if !loggedIn || a.AuthorID != current.ID {
				continue
			}
		}
		if status != "" && normalizeBlogStatus(status) != a.Status {
			continue
		}
		items = append(items, publicBlogArticle(a, false))
	}
	sort.SliceStable(items, func(i, j int) bool {
		ai, aj := asMap(items[i]), asMap(items[j])
		pi, pj := asString(ai["published_at"]), asString(aj["published_at"])
		if pi != pj {
			return pi > pj
		}
		return asString(ai["created_at"]) > asString(aj["created_at"])
	})
	writeJSON(w, http.StatusOK, map[string]any{"articles": items})
}

func (s *Server) handleCreateBlogArticle(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	var req BlogArticleRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	article, errText := buildBlogArticleFromRequest(req)
	if errText != "" {
		writeError(w, http.StatusBadRequest, errText)
		return
	}
	article.ID = randomHex(8)
	article.AuthorID = user.ID
	article.AuthorUserID = user.UserID
	article.AuthorNickname = user.Nickname
	article.CreatedAt = nowISO()
	if article.Status == "published" {
		article.PublishedAt = article.CreatedAt
	}
	store := s.loadBlogArticles()
	store.Articles = append(store.Articles, article)
	if err := s.saveBlogArticles(store); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save article.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"article": publicBlogArticle(article, true)})
}

func buildBlogArticleFromRequest(req BlogArticleRequest) (BlogArticle, string) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return BlogArticle{}, "Title is required."
	}
	if len([]rune(title)) > 120 {
		return BlogArticle{}, "Title must be 120 characters or fewer."
	}
	content := strings.TrimSpace(req.ContentMarkdown)
	if content == "" {
		return BlogArticle{}, "Content is required."
	}
	if len([]rune(content)) > 100000 {
		return BlogArticle{}, "Content is too long."
	}
	summary := strings.TrimSpace(req.Summary)
	if len([]rune(summary)) > 240 {
		summary = string([]rune(summary)[:240])
	}
	return BlogArticle{Title: title, Summary: summary, CoverURL: strings.TrimSpace(req.CoverURL), ContentMarkdown: content, Tags: normalizeBlogTags(req.Tags), Category: strings.TrimSpace(req.Category), Status: normalizeBlogStatus(req.Status)}, ""
}

func (s *Server) handleGetBlogArticle(w http.ResponseWriter, r *http.Request, id string) {
	store := s.loadBlogArticles()
	_, article := findBlogArticle(store, id)
	if article == nil {
		writeError(w, http.StatusNotFound, "Article not found.")
		return
	}
	if article.Status != "published" {
		user, ok := s.communityUserFromRequest(r)
		if !ok || !canEditBlogArticle(user, *article) {
			writeError(w, http.StatusNotFound, "Article not found.")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"article": publicBlogArticle(*article, true)})
}

func (s *Server) handleUpdateBlogArticle(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	var req BlogArticleRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	next, errText := buildBlogArticleFromRequest(req)
	if errText != "" {
		writeError(w, http.StatusBadRequest, errText)
		return
	}
	store := s.loadBlogArticles()
	idx, article := findBlogArticle(store, id)
	if article == nil {
		writeError(w, http.StatusNotFound, "Article not found.")
		return
	}
	if !canEditBlogArticle(user, *article) {
		writeError(w, http.StatusForbidden, "Only the author can edit this article.")
		return
	}
	oldStatus := article.Status
	article.Title = next.Title
	article.Summary = next.Summary
	article.CoverURL = next.CoverURL
	article.ContentMarkdown = next.ContentMarkdown
	article.Tags = next.Tags
	article.Category = next.Category
	article.Status = next.Status
	article.UpdatedAt = nowISO()
	if oldStatus != "published" && article.Status == "published" {
		article.PublishedAt = article.UpdatedAt
	}
	store.Articles[idx] = *article
	if err := s.saveBlogArticles(store); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save article.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"article": publicBlogArticle(*article, true)})
}

func (s *Server) handleDeleteBlogArticle(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	store := s.loadBlogArticles()
	idx, article := findBlogArticle(store, id)
	if article == nil {
		writeError(w, http.StatusNotFound, "Article not found.")
		return
	}
	if !canEditBlogArticle(user, *article) {
		writeError(w, http.StatusForbidden, "Only the author can delete this article.")
		return
	}
	store.Articles = append(store.Articles[:idx], store.Articles[idx+1:]...)
	if err := s.saveBlogArticles(store); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete article.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": id})
}

func (s *Server) handlePublishBlogArticle(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	store := s.loadBlogArticles()
	idx, article := findBlogArticle(store, id)
	if article == nil {
		writeError(w, http.StatusNotFound, "Article not found.")
		return
	}
	if !canEditBlogArticle(user, *article) {
		writeError(w, http.StatusForbidden, "Only the author can publish this article.")
		return
	}
	article.Status = "published"
	article.UpdatedAt = nowISO()
	if article.PublishedAt == "" {
		article.PublishedAt = article.UpdatedAt
	}
	store.Articles[idx] = *article
	if err := s.saveBlogArticles(store); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to publish article.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"article": publicBlogArticle(*article, true)})
}

func (s *Server) handleViewBlogArticle(w http.ResponseWriter, r *http.Request, id string) {
	store := s.loadBlogArticles()
	idx, article := findBlogArticle(store, id)
	if article == nil || article.Status != "published" {
		writeError(w, http.StatusNotFound, "Article not found.")
		return
	}
	article.Views++
	store.Articles[idx] = *article
	if err := s.saveBlogArticles(store); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update article views.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"views": article.Views})
}

func blogArticleIDFromPath(path string) string {
	id := strings.TrimPrefix(path, "/api/blog/articles/")
	for _, suffix := range []string{"/publish", "/view"} {
		id = strings.TrimSuffix(id, suffix)
	}
	return strings.Trim(id, "/")
}

func parsePositiveInt(raw string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return def
	}
	return n
}
