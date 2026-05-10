package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) notifyCommunityUser(userPK, typ string, payload map[string]any) {
	if s.db == nil || strings.TrimSpace(userPK) == "" {
		return
	}
	b, _ := json.Marshal(payload)
	_, _ = s.db.Exec(`INSERT INTO notifications(id, user_id, type, payload_json, created_at) VALUES(?,?,?,?,?)`, randomHex(8), userPK, typ, string(b), nowISO())
}

func (s *Server) notifyGroupMembers(groupID, exceptUserPK, typ string, payload map[string]any) {
	rows, err := s.db.Query(`SELECT user_id FROM chat_group_members WHERE group_id=? AND status='active' AND user_id!=? LIMIT 200`, groupID, exceptUserPK)
	if err != nil {
		return
	}
	userIDs := []string{}
	for rows.Next() {
		var userPK string
		if rows.Scan(&userPK) == nil {
			userIDs = append(userIDs, userPK)
		}
	}
	_ = rows.Close()
	for _, userPK := range userIDs {
		s.notifyCommunityUser(userPK, typ, payload)
	}
}

func (s *Server) handleNotifications(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	limit := parseLimit(r, 50, 100)
	rows, err := s.db.Query(`SELECT id, type, COALESCE(payload_json,'{}'), COALESCE(read_at,''), created_at FROM notifications WHERE user_id=? ORDER BY created_at DESC LIMIT ?`, user.ID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load notifications.")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, typ, payloadRaw, readAt, createdAt string
		if err := rows.Scan(&id, &typ, &payloadRaw, &readAt, &createdAt); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to read notifications.")
			return
		}
		var payload map[string]any
		_ = json.Unmarshal([]byte(payloadRaw), &payload)
		items = append(items, map[string]any{"id": id, "type": typ, "payload": payload, "read_at": readAt, "created_at": createdAt, "unread": readAt == ""})
	}
	if err := rows.Close(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read notifications.")
		return
	}
	var unread int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE user_id=? AND read_at IS NULL`, user.ID).Scan(&unread)
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "unread_count": unread})
}

func (s *Server) handleReadNotification(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	res, err := s.db.Exec(`UPDATE notifications SET read_at=COALESCE(read_at, ?) WHERE id=? AND user_id=?`, nowISO(), id, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update notification.")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeError(w, http.StatusNotFound, "Notification not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func likePattern(q string) string {
	q = strings.TrimSpace(q)
	q = strings.ReplaceAll(q, `\`, `\\`)
	q = strings.ReplaceAll(q, `%`, `\%`)
	q = strings.ReplaceAll(q, `_`, `\_`)
	return "%" + q + "%"
}

func (s *Server) handleCommunitySearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	page, pageSize, offset := parsePage(r, 20, 50)
	if q == "" {
		writeJSON(w, http.StatusOK, map[string]any{"articles": []any{}, "users": []any{}, "page": page, "page_size": pageSize, "total": 0})
		return
	}
	pat := likePattern(q)
	articles := []map[string]any{}
	rows, err := s.db.Query(`SELECT DISTINCT a.id, a.title, COALESCE(a.summary,''), COALESCE(a.cover_url,''), COALESCE(a.category,''), COALESCE(a.language,''), a.views, a.likes, a.favorites, a.comments, COALESCE(a.published_at,''), u.user_id, u.nickname
		FROM blog_articles a JOIN community_users u ON u.id=a.author_id LEFT JOIN blog_article_tags t ON t.article_id=a.id
		WHERE a.status='published' AND a.visibility='public' AND (a.title LIKE ? ESCAPE '\' OR COALESCE(a.summary,'') LIKE ? ESCAPE '\' OR COALESCE(a.content_markdown,'') LIKE ? ESCAPE '\' OR COALESCE(t.tag,'') LIKE ? ESCAPE '\' OR u.nickname LIKE ? ESCAPE '\' OR u.user_id LIKE ? ESCAPE '\')
		ORDER BY a.published_at DESC, a.created_at DESC LIMIT ? OFFSET ?`, pat, pat, pat, pat, pat, pat, pageSize, offset)
	if err == nil {
		for rows.Next() {
			var id, title, summary, cover, category, language, publishedAt, authorID, authorName string
			var views, likes, favorites, comments int
			if rows.Scan(&id, &title, &summary, &cover, &category, &language, &views, &likes, &favorites, &comments, &publishedAt, &authorID, &authorName) == nil {
				articles = append(articles, map[string]any{"id": id, "title": title, "summary": summary, "cover_url": cover, "category": category, "language": language, "views": views, "likes": likes, "favorites": favorites, "comments": comments, "published_at": publishedAt, "author": map[string]any{"id": authorID, "nickname": authorName}})
			}
		}
		_ = rows.Close()
	}
	users := []map[string]any{}
	if s.communityFrontendAllowsRequest(r) {
		userRows, err := s.db.Query(`SELECT id, user_id, nickname, COALESCE(avatar_url,''), COALESCE(bio,''), role, status, created_at, COALESCE(updated_at,''), COALESCE(last_login_at,'') FROM community_users WHERE status='active' AND (user_id LIKE ? ESCAPE '\' OR nickname LIKE ? ESCAPE '\' OR COALESCE(bio,'') LIKE ? ESCAPE '\') ORDER BY created_at DESC LIMIT ?`, pat, pat, pat, pageSize)
		if err == nil {
			defer userRows.Close()
			for userRows.Next() {
				var u CommunityUser
				if userRows.Scan(&u.ID, &u.UserID, &u.Nickname, &u.AvatarURL, &u.Bio, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt) == nil {
					users = append(users, publicCommunityUser(u))
				}
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"articles": articles, "users": users, "items": articles, "page": page, "page_size": pageSize, "has_more": len(articles) == pageSize})
}

func (s *Server) handleBlogRecommendations(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r, 10, 50)
	rows, err := s.db.Query(`SELECT a.id, a.title, COALESCE(a.summary,''), COALESCE(a.cover_url,''), COALESCE(a.category,''), COALESCE(a.language,''), a.views, a.likes, a.favorites, a.comments, COALESCE(a.published_at,''), u.user_id, u.nickname,
		(a.views + a.likes * 5 + a.favorites * 8 + a.comments * 3) AS score
		FROM blog_articles a JOIN community_users u ON u.id=a.author_id
		WHERE a.status='published' AND a.visibility='public'
		ORDER BY score DESC, a.published_at DESC, a.created_at DESC LIMIT ?`, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load recommendations.")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, title, summary, cover, category, language, publishedAt, authorID, authorName string
		var views, likes, favorites, comments, score int
		if rows.Scan(&id, &title, &summary, &cover, &category, &language, &views, &likes, &favorites, &comments, &publishedAt, &authorID, &authorName, &score) != nil {
			continue
		}
		items = append(items, map[string]any{"id": id, "title": title, "summary": summary, "cover_url": cover, "category": category, "language": language, "views": views, "likes": likes, "favorites": favorites, "comments": comments, "published_at": publishedAt, "score": score, "author": map[string]any{"id": authorID, "nickname": authorName}})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
