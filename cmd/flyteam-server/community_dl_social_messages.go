package main

import (
	"database/sql"
	"net/http"
	"sort"
	"time"
)

func (s *Server) handleFollowUser(w http.ResponseWriter, r *http.Request, targetRaw string, follow bool) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	targetID, err := s.resolveCommunityUserPK(targetRaw)
	if err != nil {
		writeError(w, http.StatusNotFound, "Target user not found.")
		return
	}
	if targetID == user.ID {
		writeError(w, http.StatusBadRequest, "You cannot follow yourself.")
		return
	}
	changed := int64(0)
	if follow {
		res, err := s.db.Exec(`INSERT OR IGNORE INTO social_follows(follower_id, following_id, created_at) VALUES(?,?,?)`, user.ID, targetID, nowISO())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to follow user.")
			return
		}
		changed, _ = res.RowsAffected()
		if changed > 0 {
			s.notifyCommunityUser(targetID, "follow", map[string]any{"actor_id": user.UserID, "actor_nickname": user.Nickname})
		}
	} else {
		res, err := s.db.Exec(`DELETE FROM social_follows WHERE follower_id=? AND following_id=?`, user.ID, targetID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to unfollow user.")
			return
		}
		changed, _ = res.RowsAffected()
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "following": follow, "changed": changed > 0, "stats": s.followStats(targetID)})
}

func (s *Server) followStats(userPK string) map[string]any {
	var followers, following int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM social_follows WHERE following_id=?`, userPK).Scan(&followers)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM social_follows WHERE follower_id=?`, userPK).Scan(&following)
	return map[string]any{"followers": followers, "following": following}
}

func (s *Server) handleFollowList(w http.ResponseWriter, r *http.Request, raw, mode string) {
	userPK, err := s.resolveCommunityUserPK(raw)
	if err != nil {
		writeError(w, http.StatusNotFound, "User not found.")
		return
	}
	page, pageSize, offset := parsePage(r, 30, 100)
	var rows *sql.Rows
	if mode == "following" {
		rows, err = s.db.Query(`SELECT u.id, u.user_id, u.nickname, COALESCE(u.avatar_url,''), COALESCE(u.bio,''), u.role, u.status, u.created_at, COALESCE(u.updated_at,''), COALESCE(u.last_login_at,'')
			FROM social_follows f JOIN community_users u ON u.id=f.following_id
			WHERE f.follower_id=? AND u.status!='deleted' ORDER BY f.created_at DESC LIMIT ? OFFSET ?`, userPK, pageSize, offset)
	} else {
		rows, err = s.db.Query(`SELECT u.id, u.user_id, u.nickname, COALESCE(u.avatar_url,''), COALESCE(u.bio,''), u.role, u.status, u.created_at, COALESCE(u.updated_at,''), COALESCE(u.last_login_at,'')
			FROM social_follows f JOIN community_users u ON u.id=f.follower_id
			WHERE f.following_id=? AND u.status!='deleted' ORDER BY f.created_at DESC LIMIT ? OFFSET ?`, userPK, pageSize, offset)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load follow list.")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var u CommunityUser
		if err := rows.Scan(&u.ID, &u.UserID, &u.Nickname, &u.AvatarURL, &u.Bio, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to read follow list.")
			return
		}
		items = append(items, publicCommunityUser(u))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "page": page, "page_size": pageSize, "has_more": len(items) == pageSize})
}

func orderedPair(a, b string) (string, string) {
	pair := []string{a, b}
	sort.Strings(pair)
	return pair[0], pair[1]
}

func (s *Server) handleMessageConversations(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	limit := parseLimit(r, 30, 100)
	rows, err := s.db.Query(`SELECT c.id, c.user_a, c.user_b, c.created_at, COALESCE(c.updated_at,''), COALESCE(c.last_message_at,''),
		COALESCE(m.content,''), COALESCE(m.created_at,'')
		FROM private_conversations c
		LEFT JOIN private_messages m ON m.id=(SELECT id FROM private_messages WHERE conversation_id=c.id AND status='normal' ORDER BY created_at DESC LIMIT 1)
		WHERE c.user_a=? OR c.user_b=?
		ORDER BY COALESCE(c.last_message_at,c.created_at) DESC LIMIT ?`, user.ID, user.ID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load conversations.")
		return
	}
	defer rows.Close()
	type convRow struct {
		id, userA, userB, createdAt, updatedAt, lastAt, lastContent, lastCreated string
	}
	rawItems := []convRow{}
	for rows.Next() {
		var item convRow
		if err := rows.Scan(&item.id, &item.userA, &item.userB, &item.createdAt, &item.updatedAt, &item.lastAt, &item.lastContent, &item.lastCreated); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to read conversations.")
			return
		}
		rawItems = append(rawItems, item)
	}
	if err := rows.Close(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read conversations.")
		return
	}
	items := []map[string]any{}
	for _, item := range rawItems {
		otherPK := item.userA
		if otherPK == user.ID {
			otherPK = item.userB
		}
		other, _ := s.loadCommunityUserByPK(otherPK)
		items = append(items, map[string]any{"id": item.id, "created_at": item.createdAt, "updated_at": item.updatedAt, "last_message_at": item.lastAt, "last_message": map[string]any{"content": item.lastContent, "created_at": item.lastCreated}, "other_user": publicCommunityUser(other), "unread_count": s.privateUnreadCount(item.id, user.ID)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleCreateMessageConversation(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	var req conversationCreateRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	targetRaw := firstNonEmpty(req.TargetUserID, req.UserID, req.RecipientID)
	targetID, err := s.resolveCommunityUserPK(targetRaw)
	if err != nil {
		writeError(w, http.StatusNotFound, "Target user not found.")
		return
	}
	if targetID == user.ID {
		writeError(w, http.StatusBadRequest, "Cannot create a conversation with yourself.")
		return
	}
	userA, userB := orderedPair(user.ID, targetID)
	id := randomHex(8)
	now := nowISO()
	_, _ = s.db.Exec(`INSERT OR IGNORE INTO private_conversations(id, user_a, user_b, created_at, updated_at) VALUES(?,?,?,?,?)`, id, userA, userB, now, now)
	var conversationID string
	if err := s.db.QueryRow(`SELECT id FROM private_conversations WHERE user_a=? AND user_b=?`, userA, userB).Scan(&conversationID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create conversation.")
		return
	}
	s.handleMessageConversation(w, r, conversationID)
}

func (s *Server) conversationForUser(conversationID, userPK string) (map[string]any, string, bool) {
	var id, userA, userB, createdAt, updatedAt, lastAt string
	err := s.db.QueryRow(`SELECT id, user_a, user_b, created_at, COALESCE(updated_at,''), COALESCE(last_message_at,'') FROM private_conversations WHERE id=? AND (user_a=? OR user_b=?)`, conversationID, userPK, userPK).Scan(&id, &userA, &userB, &createdAt, &updatedAt, &lastAt)
	if err != nil {
		return nil, "", false
	}
	otherPK := userA
	if otherPK == userPK {
		otherPK = userB
	}
	other, _ := s.loadCommunityUserByPK(otherPK)
	return map[string]any{"id": id, "created_at": createdAt, "updated_at": updatedAt, "last_message_at": lastAt, "other_user": publicCommunityUser(other), "unread_count": s.privateUnreadCount(id, userPK)}, otherPK, true
}

func (s *Server) handleMessageConversation(w http.ResponseWriter, r *http.Request, conversationID string) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	conv, _, ok := s.conversationForUser(conversationID, user.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "Conversation not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversation": conv})
}

func (s *Server) privateUnreadCount(conversationID, userPK string) int {
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM private_messages WHERE conversation_id=? AND sender_id!=? AND read_at IS NULL AND status='normal'`, conversationID, userPK).Scan(&n)
	return n
}

func (s *Server) handleConversationMessages(w http.ResponseWriter, r *http.Request, conversationID string) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	if _, _, ok := s.conversationForUser(conversationID, user.ID); !ok {
		writeError(w, http.StatusNotFound, "Conversation not found.")
		return
	}
	limit := parseLimit(r, 50, 100)
	rows, err := s.db.Query(`SELECT m.id, m.sender_id, m.content, m.created_at, COALESCE(m.read_at,''), u.user_id, u.nickname
		FROM private_messages m JOIN community_users u ON u.id=m.sender_id
		WHERE m.conversation_id=? AND m.status='normal'
		ORDER BY m.created_at DESC LIMIT ?`, conversationID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load messages.")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, senderPK, content, createdAt, readAt, senderID, nickname string
		if err := rows.Scan(&id, &senderPK, &content, &createdAt, &readAt, &senderID, &nickname); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to read messages.")
			return
		}
		items = append(items, map[string]any{"id": id, "content": content, "created_at": createdAt, "read_at": readAt, "sender": map[string]any{"id": senderID, "user_pk": senderPK, "nickname": nickname}, "mine": senderPK == user.ID})
	}
	if err := rows.Close(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read messages.")
		return
	}
	reverseMaps(items)
	_, _ = s.db.Exec(`UPDATE private_messages SET read_at=? WHERE conversation_id=? AND sender_id!=? AND read_at IS NULL`, nowISO(), conversationID, user.ID)
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func reverseMaps(items []map[string]any) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}

func (s *Server) handleSendConversationMessage(w http.ResponseWriter, r *http.Request, conversationID string) {
	user, ok := s.requireCommunityWriter(w, r)
	if !ok {
		return
	}
	conv, otherPK, ok := s.conversationForUser(conversationID, user.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "Conversation not found.")
		return
	}
	if !s.checkRateLimit("pm:"+user.ID+":"+clientIP(r), 60, 10*time.Minute, true) {
		writeError(w, http.StatusTooManyRequests, "Message requests are too frequent.")
		return
	}
	var req messageCreateRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	content, err := cleanCommunityText(req.Content, 4000)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id := randomHex(8)
	now := nowISO()
	if _, err := s.db.Exec(`INSERT INTO private_messages(id, conversation_id, sender_id, content, status, created_at) VALUES(?,?,?,?,'normal',?)`, id, conversationID, user.ID, content, now); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to send message.")
		return
	}
	_, _ = s.db.Exec(`UPDATE private_conversations SET updated_at=?, last_message_at=? WHERE id=?`, now, now, conversationID)
	s.notifyCommunityUser(otherPK, "private_message", map[string]any{"conversation_id": conversationID, "message_id": id, "actor_id": user.UserID, "actor_nickname": user.Nickname, "preview": previewText(content, 80)})
	writeJSON(w, http.StatusOK, map[string]any{"message": map[string]any{"id": id, "conversation_id": conversationID, "content": content, "created_at": now, "sender": publicCommunityUser(user)}, "conversation": conv})
}
