package main

import (
	"database/sql"
	"net/http"
	"sort"
	"strings"
)

type BlogArticleRequest struct {
	Title           string   `json:"title"`
	Summary         string   `json:"summary"`
	CoverURL        string   `json:"cover_url"`
	ContentMarkdown string   `json:"content_markdown"`
	Tags            []string `json:"tags"`
	Category        string   `json:"category"`
	Language        string   `json:"language"`
	Status          string   `json:"status"`
}

type BlogArticle struct {
	ID              string
	AuthorID        string
	AuthorUserID    string
	AuthorNickname  string
	Title           string
	Summary         string
	CoverURL        string
	ContentMarkdown string
	Status          string
	Visibility      string
	Language        string
	Category        string
	Views           int
	Likes           int
	Favorites       int
	Comments        int
	CreatedAt       string
	UpdatedAt       string
	PublishedAt     string
	Tags            []string
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
		key := strings.ToLower(clean)
		if clean == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, clean)
		if len(out) >= 8 {
			break
		}
	}
	return out
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
	return BlogArticle{Title: title, Summary: summary, CoverURL: strings.TrimSpace(req.CoverURL), ContentMarkdown: content, Tags: normalizeBlogTags(req.Tags), Category: strings.TrimSpace(req.Category), Language: strings.TrimSpace(req.Language), Status: normalizeBlogStatus(req.Status), Visibility: "public"}, ""
}

func publicBlogArticle(a BlogArticle, includeContent bool) map[string]any {
	out := map[string]any{
		"id":              a.ID,
		"author_id":       a.AuthorID,
		"author_user_id":  a.AuthorUserID,
		"author_nickname": a.AuthorNickname,
		"title":           a.Title,
		"summary":         a.Summary,
		"cover_url":       a.CoverURL,
		"tags":            a.Tags,
		"category":        a.Category,
		"language":        a.Language,
		"status":          a.Status,
		"visibility":      a.Visibility,
		"views":           a.Views,
		"likes":           a.Likes,
		"favorites":       a.Favorites,
		"comments":        a.Comments,
		"created_at":      a.CreatedAt,
		"updated_at":      a.UpdatedAt,
		"published_at":    a.PublishedAt,
	}
	if includeContent {
		out["content_markdown"] = a.ContentMarkdown
	}
	return out
}

func (s *Server) loadBlogArticleTags(articleID string) []string {
	rows, err := s.db.Query(`SELECT tag FROM blog_article_tags WHERE article_id=? ORDER BY tag ASC`, articleID)
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

func (s *Server) scanBlogArticleRow(scanner interface{ Scan(...any) error }) (BlogArticle, error) {
	var a BlogArticle
	err := scanner.Scan(&a.ID, &a.AuthorID, &a.AuthorUserID, &a.AuthorNickname, &a.Title, &a.Summary, &a.CoverURL, &a.ContentMarkdown, &a.Status, &a.Visibility, &a.Language, &a.Category, &a.Views, &a.Likes, &a.Favorites, &a.Comments, &a.PublishedAt, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

func (s *Server) loadBlogArticleByID(id string) (BlogArticle, error) {
	row := s.db.QueryRow(`SELECT a.id, a.author_id, COALESCE(u.user_id,''), COALESCE(u.nickname,''), a.title, COALESCE(a.summary,''), COALESCE(a.cover_url,''), a.content_markdown, a.status, a.visibility, COALESCE(a.language,''), COALESCE(a.category,''), a.views, a.likes, a.favorites, a.comments, COALESCE(a.published_at,''), a.created_at, COALESCE(a.updated_at,'')
		FROM blog_articles a LEFT JOIN community_users u ON u.id=a.author_id WHERE a.id=?`, id)
	a, err := s.scanBlogArticleRow(row)
	if err != nil {
		return BlogArticle{}, err
	}
	a.Tags = s.loadBlogArticleTags(a.ID)
	return a, nil
}

func canEditBlogArticle(user CommunityUser, article BlogArticle) bool {
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
	current, _, loggedIn := s.communityUserFromRequest(r)
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	mine := r.URL.Query().Get("mine") == "1" || strings.EqualFold(r.URL.Query().Get("mine"), "true")
	page, pageSize, offset := parsePage(r, 20, 100)
	where := []string{}
	args := []any{}
	if mine {
		if !loggedIn {
			writeError(w, http.StatusUnauthorized, "User login required.")
			return
		}
		where = append(where, "a.author_id=?")
		args = append(args, current.ID)
	} else {
		where = append(where, "a.status='published'", "a.visibility='public'")
	}
	if status != "" {
		where = append(where, "a.status=?")
		args = append(args, normalizeBlogStatus(status))
	}
	whereSQL := strings.Join(where, " AND ")
	var total int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM blog_articles a WHERE `+whereSQL, args...).Scan(&total)
	args = append(args, pageSize, offset)
	rows, err := s.db.Query(`SELECT a.id, a.author_id, COALESCE(u.user_id,''), COALESCE(u.nickname,''), a.title, COALESCE(a.summary,''), COALESCE(a.cover_url,''), a.content_markdown, a.status, a.visibility, COALESCE(a.language,''), COALESCE(a.category,''), a.views, a.likes, a.favorites, a.comments, COALESCE(a.published_at,''), a.created_at, COALESCE(a.updated_at,'')
		FROM blog_articles a LEFT JOIN community_users u ON u.id=a.author_id WHERE `+whereSQL+` ORDER BY COALESCE(a.published_at,a.created_at) DESC, a.created_at DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load articles.")
		return
	}
	raw := []BlogArticle{}
	for rows.Next() {
		a, err := s.scanBlogArticleRow(rows)
		if err != nil {
			_ = rows.Close()
			writeError(w, http.StatusInternalServerError, "Failed to read articles.")
			return
		}
		raw = append(raw, a)
	}
	if err := rows.Close(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read articles.")
		return
	}
	items := []map[string]any{}
	for _, a := range raw {
		a.Tags = s.loadBlogArticleTags(a.ID)
		items = append(items, publicBlogArticle(a, false))
	}
	writeJSON(w, http.StatusOK, map[string]any{"articles": items, "items": items, "page": page, "page_size": pageSize, "total": total, "has_more": offset+len(items) < total})
}

func (s *Server) handleCreateBlogArticle(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireCommunityWriter(w, r)
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
	if err := s.insertOrUpdateBlogArticle(article, user.ID, true); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save article.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"article": publicBlogArticle(article, true)})
}

func (s *Server) insertOrUpdateBlogArticle(a BlogArticle, actorID string, createVersion bool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	publishedAt := sql.NullString{String: strings.TrimSpace(a.PublishedAt), Valid: strings.TrimSpace(a.PublishedAt) != ""}
	updatedAt := sql.NullString{String: strings.TrimSpace(a.UpdatedAt), Valid: strings.TrimSpace(a.UpdatedAt) != ""}
	_, err = tx.Exec(`INSERT INTO blog_articles(id, author_id, title, summary, cover_url, content_markdown, status, visibility, language, category, views, published_at, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET title=excluded.title, summary=excluded.summary, cover_url=excluded.cover_url, content_markdown=excluded.content_markdown, status=excluded.status, visibility=excluded.visibility, language=excluded.language, category=excluded.category, published_at=excluded.published_at, updated_at=excluded.updated_at`, a.ID, a.AuthorID, a.Title, a.Summary, a.CoverURL, a.ContentMarkdown, a.Status, a.Visibility, a.Language, a.Category, a.Views, publishedAt, a.CreatedAt, updatedAt)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM blog_article_tags WHERE article_id=?`, a.ID); err != nil {
		return err
	}
	for _, tag := range a.Tags {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO blog_article_tags(article_id, tag) VALUES(?,?)`, a.ID, tag); err != nil {
			return err
		}
	}
	if createVersion {
		_, _ = tx.Exec(`INSERT INTO blog_article_versions(id, article_id, title, summary, content_markdown, created_at, created_by) VALUES(?,?,?,?,?,?,?)`, randomHex(8), a.ID, a.Title, a.Summary, a.ContentMarkdown, nowISO(), actorID)
	}
	return tx.Commit()
}

func (s *Server) handleGetBlogArticle(w http.ResponseWriter, r *http.Request, id string) {
	article, err := s.loadBlogArticleByID(id)
	if err != nil || article.Status == "deleted" {
		writeError(w, http.StatusNotFound, "Article not found.")
		return
	}
	if article.Status != "published" || article.Visibility != "public" {
		user, _, ok := s.communityUserFromRequest(r)
		if !ok || (!canEditBlogArticle(user, article) && !s.canModerateCommunity(r, user)) {
			writeError(w, http.StatusNotFound, "Article not found.")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"article": publicBlogArticle(article, true)})
}

func (s *Server) handleUpdateBlogArticle(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := s.requireCommunityWriter(w, r)
	if !ok {
		return
	}
	old, err := s.loadBlogArticleByID(id)
	if err != nil || old.Status == "deleted" {
		writeError(w, http.StatusNotFound, "Article not found.")
		return
	}
	if !canEditBlogArticle(user, old) && !s.canModerateCommunity(r, user) {
		writeError(w, http.StatusForbidden, "Only the author can edit this article.")
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
	old.Title = next.Title
	old.Summary = next.Summary
	old.CoverURL = next.CoverURL
	old.ContentMarkdown = next.ContentMarkdown
	old.Tags = next.Tags
	old.Category = next.Category
	old.Language = next.Language
	old.Visibility = "public"
	old.UpdatedAt = nowISO()
	prevStatus := old.Status
	old.Status = next.Status
	if prevStatus != "published" && old.Status == "published" {
		old.PublishedAt = old.UpdatedAt
	}
	if err := s.insertOrUpdateBlogArticle(old, user.ID, true); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save article.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"article": publicBlogArticle(old, true)})
}

func (s *Server) handleDeleteBlogArticle(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	article, err := s.loadBlogArticleByID(id)
	if err != nil || article.Status == "deleted" {
		writeError(w, http.StatusNotFound, "Article not found.")
		return
	}
	if !canEditBlogArticle(user, article) && !s.canModerateCommunity(r, user) {
		writeError(w, http.StatusForbidden, "Only the author can delete this article.")
		return
	}
	_, err = s.db.Exec(`UPDATE blog_articles SET status='deleted', updated_at=? WHERE id=?`, nowISO(), id)
	if err != nil {
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
	article, err := s.loadBlogArticleByID(id)
	if err != nil || article.Status == "deleted" {
		writeError(w, http.StatusNotFound, "Article not found.")
		return
	}
	if !canEditBlogArticle(user, article) && !s.canModerateCommunity(r, user) {
		writeError(w, http.StatusForbidden, "Only the author can publish this article.")
		return
	}
	article.Status = "published"
	article.UpdatedAt = nowISO()
	if article.PublishedAt == "" {
		article.PublishedAt = article.UpdatedAt
	}
	if err := s.insertOrUpdateBlogArticle(article, user.ID, true); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to publish article.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"article": publicBlogArticle(article, true)})
}

func (s *Server) handleViewBlogArticle(w http.ResponseWriter, r *http.Request, id string) {
	article, err := s.loadBlogArticleByID(id)
	if err != nil || article.Status != "published" || article.Visibility != "public" {
		writeError(w, http.StatusNotFound, "Article not found.")
		return
	}
	_, _ = s.db.Exec(`UPDATE blog_articles SET views=views+1 WHERE id=?`, id)
	article.Views++
	writeJSON(w, http.StatusOK, map[string]any{"views": article.Views})
}

func blogArticleIDFromPath(path string) string {
	id := strings.TrimPrefix(path, "/api/blog/articles/")
	for _, suffix := range []string{"/publish", "/view", "/comments", "/like", "/favorite"} {
		id = strings.TrimSuffix(id, suffix)
	}
	return strings.Trim(id, "/")
}

func sortBlogArticles(items []map[string]any) {
	sort.SliceStable(items, func(i, j int) bool {
		pi, pj := asString(items[i]["published_at"]), asString(items[j]["published_at"])
		if pi != pj {
			return pi > pj
		}
		return asString(items[i]["created_at"]) > asString(items[j]["created_at"])
	})
}
