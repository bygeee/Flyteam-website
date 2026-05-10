package app

import (
	"database/sql"
	"net/http"
	"strings"
)

type friendRequestCreateRequest struct {
	TargetUserID string `json:"target_user_id"`
	UserID       string `json:"user_id"`
	Message      string `json:"message"`
}

func friendshipPair(a, b string) (string, string) { return orderedPair(a, b) }

func (s *Server) areFriends(a, b string) bool {
	if strings.TrimSpace(a) == "" || strings.TrimSpace(b) == "" || a == b || s.db == nil {
		return false
	}
	ua, ub := friendshipPair(a, b)
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM friendships WHERE user_a=? AND user_b=?`, ua, ub).Scan(&n)
	return n > 0
}

func (s *Server) insertFriendshipTx(tx *sql.Tx, a, b string) error {
	ua, ub := friendshipPair(a, b)
	_, err := tx.Exec(`INSERT OR IGNORE INTO friendships(user_a, user_b, created_at) VALUES(?,?,?)`, ua, ub, nowISO())
	return err
}

func (s *Server) handleListFriends(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	rows, err := s.db.Query(`SELECT u.id, u.user_id, u.nickname, COALESCE(u.avatar_url,''), COALESCE(u.bio,''), u.role, u.status, u.created_at, COALESCE(u.updated_at,''), COALESCE(u.last_login_at,''), f.created_at
		FROM friendships f JOIN community_users u ON u.id = CASE WHEN f.user_a=? THEN f.user_b ELSE f.user_a END
		WHERE (f.user_a=? OR f.user_b=?) AND u.status!='deleted' ORDER BY f.created_at DESC`, user.ID, user.ID, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load friends.")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var u CommunityUser
		var since string
		if err := rows.Scan(&u.ID, &u.UserID, &u.Nickname, &u.AvatarURL, &u.Bio, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt, &since); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to read friends.")
			return
		}
		m := publicCommunityUser(u)
		m["friend_since"] = since
		items = append(items, m)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

func (s *Server) handleCreateFriendRequest(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	var req friendRequestCreateRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	targetRaw := firstNonEmpty(req.TargetUserID, req.UserID)
	targetID, err := s.resolveCommunityUserPK(targetRaw)
	if err != nil {
		writeError(w, http.StatusNotFound, "Target user not found.")
		return
	}
	if targetID == user.ID {
		writeError(w, http.StatusBadRequest, "You cannot add yourself as a friend.")
		return
	}
	if s.areFriends(user.ID, targetID) {
		writeError(w, http.StatusConflict, "Already friends.")
		return
	}
	message := strings.TrimSpace(req.Message)
	if len([]rune(message)) > 120 {
		message = string([]rune(message)[:120])
	}

	var id, requester, addressee string
	err = s.db.QueryRow(`SELECT id, requester_id, addressee_id FROM friend_requests WHERE status='pending' AND ((requester_id=? AND addressee_id=?) OR (requester_id=? AND addressee_id=?)) ORDER BY created_at DESC LIMIT 1`, user.ID, targetID, targetID, user.ID).Scan(&id, &requester, &addressee)
	if err == nil {
		if requester == user.ID {
			writeError(w, http.StatusConflict, "Friend request already sent.")
			return
		}
		tx, err := s.db.Begin()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Database error.")
			return
		}
		defer tx.Rollback()
		now := nowISO()
		if _, err := tx.Exec(`UPDATE friend_requests SET status='accepted', updated_at=? WHERE id=?`, now, id); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to accept reverse request.")
			return
		}
		if err := s.insertFriendshipTx(tx, user.ID, targetID); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to save friendship.")
			return
		}
		if err := tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to save friendship.")
			return
		}
		s.notifyCommunityUser(targetID, "friend_accept", map[string]any{"actor_id": user.UserID, "actor_nickname": user.Nickname})
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "accepted_reverse_request": true, "request_id": id})
		return
	}
	if err != sql.ErrNoRows {
		writeError(w, http.StatusInternalServerError, "Failed to check friend requests.")
		return
	}
	id = randomHex(8)
	now := nowISO()
	if _, err := s.db.Exec(`INSERT INTO friend_requests(id, requester_id, addressee_id, message, status, created_at) VALUES(?,?,?,?, 'pending', ?)`, id, user.ID, targetID, message, now); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to send friend request.")
		return
	}
	s.notifyCommunityUser(targetID, "friend_request", map[string]any{"request_id": id, "actor_id": user.UserID, "actor_nickname": user.Nickname, "message": message})
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "request": map[string]any{"id": id, "status": "pending", "message": message, "created_at": now}})
}

func (s *Server) handleFriendRequests(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	box := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("box")))
	where := "(fr.requester_id=? OR fr.addressee_id=?)"
	args := []any{user.ID, user.ID}
	if box == "inbox" {
		where = "fr.addressee_id=?"
		args = []any{user.ID}
	} else if box == "outbox" {
		where = "fr.requester_id=?"
		args = []any{user.ID}
	}
	rows, err := s.db.Query(`SELECT fr.id, fr.requester_id, fr.addressee_id, COALESCE(fr.message,''), fr.status, fr.created_at, COALESCE(fr.updated_at,''),
		u.id, u.user_id, u.nickname, COALESCE(u.avatar_url,''), COALESCE(u.bio,''), u.role, u.status, u.created_at, COALESCE(u.updated_at,''), COALESCE(u.last_login_at,'')
		FROM friend_requests fr JOIN community_users u ON u.id = CASE WHEN fr.requester_id=? THEN fr.addressee_id ELSE fr.requester_id END
		WHERE `+where+` ORDER BY fr.created_at DESC LIMIT 100`, append([]any{user.ID}, args...)...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load friend requests.")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, requester, addressee, message, status, createdAt, updatedAt string
		var u CommunityUser
		if err := rows.Scan(&id, &requester, &addressee, &message, &status, &createdAt, &updatedAt, &u.ID, &u.UserID, &u.Nickname, &u.AvatarURL, &u.Bio, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to read friend requests.")
			return
		}
		direction := "outbox"
		if addressee == user.ID {
			direction = "inbox"
		}
		items = append(items, map[string]any{"id": id, "status": status, "direction": direction, "message": message, "created_at": createdAt, "updated_at": updatedAt, "other_user": publicCommunityUser(u)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleFriendRequestAction(w http.ResponseWriter, r *http.Request, requestID, action string) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	var requester, addressee, status string
	if err := s.db.QueryRow(`SELECT requester_id, addressee_id, status FROM friend_requests WHERE id=?`, requestID).Scan(&requester, &addressee, &status); err != nil {
		writeError(w, http.StatusNotFound, "Friend request not found.")
		return
	}
	if status != "pending" {
		writeError(w, http.StatusConflict, "Friend request is not pending.")
		return
	}
	if action == "accept" {
		if addressee != user.ID {
			writeError(w, http.StatusForbidden, "Only the receiver can accept this request.")
			return
		}
		tx, err := s.db.Begin()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Database error.")
			return
		}
		defer tx.Rollback()
		now := nowISO()
		if _, err := tx.Exec(`UPDATE friend_requests SET status='accepted', updated_at=? WHERE id=?`, now, requestID); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to accept friend request.")
			return
		}
		if err := s.insertFriendshipTx(tx, requester, addressee); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to save friendship.")
			return
		}
		if err := tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to accept friend request.")
			return
		}
		s.notifyCommunityUser(requester, "friend_accept", map[string]any{"request_id": requestID, "actor_id": user.UserID, "actor_nickname": user.Nickname})
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": "accepted"})
		return
	}
	if requester != user.ID && addressee != user.ID {
		writeError(w, http.StatusForbidden, "No permission to update this request.")
		return
	}
	next := "rejected"
	if requester == user.ID {
		next = "canceled"
	}
	if _, err := s.db.Exec(`UPDATE friend_requests SET status=?, updated_at=? WHERE id=?`, next, nowISO(), requestID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update friend request.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": next})
}

func (s *Server) handleRemoveFriend(w http.ResponseWriter, r *http.Request, raw string) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	targetID, err := s.resolveCommunityUserPK(raw)
	if err != nil {
		writeError(w, http.StatusNotFound, "Friend not found.")
		return
	}
	ua, ub := friendshipPair(user.ID, targetID)
	res, err := s.db.Exec(`DELETE FROM friendships WHERE user_a=? AND user_b=?`, ua, ub)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to remove friend.")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeError(w, http.StatusNotFound, "Friend not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
