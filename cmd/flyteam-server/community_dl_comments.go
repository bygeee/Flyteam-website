package main

import (
	"database/sql"
	"net/http"
	"time"
)

func (s *Server) refreshArticleStatsTx(tx *sql.Tx, articleID string) error {
	var likes, favorites, comments int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM blog_likes WHERE article_id=?`, articleID).Scan(&likes); err != nil {
		return err
	}
	if err := tx.QueryRow(`SELECT COUNT(*) FROM blog_favorites WHERE article_id=?`, articleID).Scan(&favorites); err != nil {
		return err
	}
	if err := tx.QueryRow(`SELECT COUNT(*) FROM blog_comments WHERE article_id=? AND status='visible'`, articleID).Scan(&comments); err != nil {
		return err
	}
	_, err := tx.Exec(`UPDATE blog_articles SET likes=?, favorites=?, comments=? WHERE id=?`, likes, favorites, comments, articleID)
	return err
}

func (s *Server) articleStats(articleID string) map[string]any {
	var likes, favorites, comments int
	_ = s.db.QueryRow(`SELECT likes, favorites, comments FROM blog_articles WHERE id=?`, articleID).Scan(&likes, &favorites, &comments)
	return map[string]any{"likes": likes, "favorites": favorites, "comments": comments}
}

func (s *Server) handleArticleComments(w http.ResponseWriter, r *http.Request, articleID string) {
	if !s.communityFrontendAllowsRequest(r) {
		s.handleCommunityLoginRequired(w, r)
		return
	}
	article, err := s.loadArticleMeta(articleID)
	if err != nil || !articleReadable(article) {
		writeError(w, http.StatusNotFound, "Article not found.")
		return
	}
	page, pageSize, offset := parsePage(r, 20, 100)
	var total int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM blog_comments WHERE article_id=? AND status='visible'`, article.ID).Scan(&total)
	rows, err := s.db.Query(`SELECT c.id, COALESCE(c.parent_id,''), c.content, c.created_at, COALESCE(c.updated_at,''), u.id, u.user_id, u.nickname, COALESCE(u.avatar_url,'')
		FROM blog_comments c JOIN community_users u ON u.id=c.author_id
		WHERE c.article_id=? AND c.status='visible'
		ORDER BY c.created_at ASC LIMIT ? OFFSET ?`, article.ID, pageSize, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load comments.")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, parentID, content, createdAt, updatedAt, authorPK, userID, nickname, avatar string
		if err := rows.Scan(&id, &parentID, &content, &createdAt, &updatedAt, &authorPK, &userID, &nickname, &avatar); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to read comments.")
			return
		}
		items = append(items, map[string]any{"id": id, "article_id": article.ID, "parent_id": parentID, "content": content, "created_at": createdAt, "updated_at": updatedAt, "author": map[string]any{"id": userID, "user_pk": authorPK, "nickname": nickname, "avatar_url": avatar}})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "comments": items, "page": page, "page_size": pageSize, "total": total, "has_more": offset+len(items) < total})
}

func (s *Server) handleAddArticleComment(w http.ResponseWriter, r *http.Request, articleID string) {
	user, ok := s.requireCommunityWriter(w, r)
	if !ok {
		return
	}
	if !s.checkRateLimit("comment:"+user.ID+":"+clientIP(r), 30, 10*time.Minute, true) {
		writeError(w, http.StatusTooManyRequests, "Comment requests are too frequent.")
		return
	}
	article, err := s.loadArticleMeta(articleID)
	if err != nil || !articleInteractive(article) {
		writeError(w, http.StatusNotFound, "Article not found.")
		return
	}
	var req commentRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	content, err := cleanCommunityText(req.Content, 2000)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	parentID := req.ParentID
	if parentID != "" {
		var exists int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM blog_comments WHERE id=? AND article_id=? AND status='visible'`, parentID, article.ID).Scan(&exists)
		if exists == 0 {
			writeError(w, http.StatusBadRequest, "Parent comment not found.")
			return
		}
	}
	id := randomHex(8)
	now := nowISO()
	tx, err := s.db.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error.")
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO blog_comments(id, article_id, author_id, parent_id, content, status, created_at) VALUES(?,?,?,?,?,'visible',?)`, id, article.ID, user.ID, nullIfEmpty(parentID), content, now); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save comment.")
		return
	}
	if err := s.refreshArticleStatsTx(tx, article.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update comment counter.")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save comment.")
		return
	}
	if article.AuthorID != user.ID {
		s.notifyCommunityUser(article.AuthorID, "comment", map[string]any{"article_id": article.ID, "article_title": article.Title, "comment_id": id, "actor_id": user.UserID, "actor_nickname": user.Nickname})
	}
	writeJSON(w, http.StatusOK, map[string]any{"comment": map[string]any{"id": id, "article_id": article.ID, "parent_id": parentID, "content": content, "created_at": now, "author": publicCommunityUser(user)}, "stats": s.articleStats(article.ID)})
}

func (s *Server) loadCommentOwner(commentID string) (articleID, authorID, status string, err error) {
	err = s.db.QueryRow(`SELECT article_id, author_id, status FROM blog_comments WHERE id=?`, commentID).Scan(&articleID, &authorID, &status)
	return
}

func (s *Server) handleUpdateBlogComment(w http.ResponseWriter, r *http.Request, commentID string) {
	user, _, _ := s.communityUserFromRequest(r)
	articleID, authorID, status, err := s.loadCommentOwner(commentID)
	if err != nil || status == "deleted" {
		writeError(w, http.StatusNotFound, "Comment not found.")
		return
	}
	if user.ID == "" && !s.canModerateCommunity(r, user) {
		writeError(w, http.StatusUnauthorized, "User login required.")
		return
	}
	if user.ID != authorID && !s.canModerateCommunity(r, user) {
		writeError(w, http.StatusForbidden, "You can only edit your own comments.")
		return
	}
	if user.Status == "muted" && !s.canModerateCommunity(r, user) {
		writeError(w, http.StatusForbidden, "This user is muted.")
		return
	}
	var req commentRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	content, err := cleanCommunityText(req.Content, 2000)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	_, err = s.db.Exec(`UPDATE blog_comments SET content=?, updated_at=? WHERE id=?`, content, nowISO(), commentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update comment.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "article_id": articleID})
}

func (s *Server) handleDeleteBlogComment(w http.ResponseWriter, r *http.Request, commentID string) {
	user, _, _ := s.communityUserFromRequest(r)
	articleID, authorID, status, err := s.loadCommentOwner(commentID)
	if err != nil || status == "deleted" {
		writeError(w, http.StatusNotFound, "Comment not found.")
		return
	}
	if user.ID == "" && !s.canModerateCommunity(r, user) {
		writeError(w, http.StatusUnauthorized, "User login required.")
		return
	}
	if user.ID != authorID && !s.canModerateCommunity(r, user) {
		writeError(w, http.StatusForbidden, "You can only delete your own comments.")
		return
	}
	tx, err := s.db.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error.")
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE blog_comments SET status='deleted', updated_at=? WHERE id=?`, nowISO(), commentID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete comment.")
		return
	}
	if err := s.refreshArticleStatsTx(tx, articleID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update comment counter.")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete comment.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "stats": s.articleStats(articleID)})
}

func (s *Server) handleSetArticleReaction(w http.ResponseWriter, r *http.Request, articleID, kind string, set bool) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	article, err := s.loadArticleMeta(articleID)
	if err != nil || !articleInteractive(article) {
		writeError(w, http.StatusNotFound, "Article not found.")
		return
	}
	table := "blog_likes"
	field := "liked"
	if kind == "favorite" {
		table = "blog_favorites"
		field = "favorited"
	}
	tx, err := s.db.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error.")
		return
	}
	defer tx.Rollback()
	changed := int64(0)
	if set {
		res, err := tx.Exec(`INSERT OR IGNORE INTO `+table+`(article_id, user_id, created_at) VALUES(?,?,?)`, article.ID, user.ID, nowISO())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to update reaction.")
			return
		}
		changed, _ = res.RowsAffected()
	} else {
		res, err := tx.Exec(`DELETE FROM `+table+` WHERE article_id=? AND user_id=?`, article.ID, user.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to update reaction.")
			return
		}
		changed, _ = res.RowsAffected()
	}
	if err := s.refreshArticleStatsTx(tx, article.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update counters.")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update reaction.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, field: set, "changed": changed > 0, "stats": s.articleStats(article.ID)})
}
