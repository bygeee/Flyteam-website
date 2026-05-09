package main

import (
	"net/http"
	"strings"
	"time"
)

func (s *Server) handleGroups(w http.ResponseWriter, r *http.Request) {
	user, _, _ := s.communityUserFromRequest(r)
	page, pageSize, offset := parsePage(r, 20, 100)
	args := []any{pageSize, offset}
	where := `g.visibility='public'`
	if user.ID != "" {
		where = `(g.visibility='public' OR EXISTS(SELECT 1 FROM chat_group_members gm WHERE gm.group_id=g.id AND gm.user_id=? AND gm.status='active'))`
		args = []any{user.ID, pageSize, offset}
	}
	rows, err := s.db.Query(`SELECT g.id, g.owner_id, g.name, COALESCE(g.avatar_url,''), COALESCE(g.intro,''), g.visibility, g.created_at, COALESCE(g.updated_at,''),
		(SELECT COUNT(*) FROM chat_group_members gm WHERE gm.group_id=g.id AND gm.status='active') AS member_count
		FROM chat_groups g WHERE `+where+` ORDER BY g.created_at DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load groups.")
		return
	}
	defer rows.Close()
	type groupRow struct {
		id, ownerID, name, avatar, intro, visibility, createdAt, updatedAt string
		memberCount                                                        int
	}
	rawItems := []groupRow{}
	for rows.Next() {
		var item groupRow
		if err := rows.Scan(&item.id, &item.ownerID, &item.name, &item.avatar, &item.intro, &item.visibility, &item.createdAt, &item.updatedAt, &item.memberCount); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to read groups.")
			return
		}
		rawItems = append(rawItems, item)
	}
	if err := rows.Close(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read groups.")
		return
	}
	items := []map[string]any{}
	for _, g := range rawItems {
		owner, _ := s.loadCommunityUserByPK(g.ownerID)
		role, memberStatus := s.groupMemberRole(g.id, user.ID)
		items = append(items, map[string]any{"id": g.id, "name": g.name, "avatar_url": g.avatar, "intro": g.intro, "visibility": g.visibility, "created_at": g.createdAt, "updated_at": g.updatedAt, "owner": publicCommunityUser(owner), "member_count": g.memberCount, "my_role": role, "my_status": memberStatus})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "page": page, "page_size": pageSize, "has_more": len(items) == pageSize})
}

func (s *Server) scanGroupRow(rows interface{ Scan(...any) error }, viewerPK string) (map[string]any, error) {
	var id, ownerID, name, avatar, intro, visibility, createdAt, updatedAt string
	var memberCount int
	if err := rows.Scan(&id, &ownerID, &name, &avatar, &intro, &visibility, &createdAt, &updatedAt, &memberCount); err != nil {
		return nil, err
	}
	owner, _ := s.loadCommunityUserByPK(ownerID)
	role, memberStatus := s.groupMemberRole(id, viewerPK)
	return map[string]any{"id": id, "name": name, "avatar_url": avatar, "intro": intro, "visibility": visibility, "created_at": createdAt, "updated_at": updatedAt, "owner": publicCommunityUser(owner), "member_count": memberCount, "my_role": role, "my_status": memberStatus}, nil
}

func (s *Server) groupMemberRole(groupID, userPK string) (role, status string) {
	if groupID == "" || userPK == "" {
		return "", ""
	}
	_ = s.db.QueryRow(`SELECT role, status FROM chat_group_members WHERE group_id=? AND user_id=?`, groupID, userPK).Scan(&role, &status)
	return
}

func (s *Server) requireActiveGroupMember(w http.ResponseWriter, r *http.Request, groupID string) (CommunityUser, string, bool) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return CommunityUser{}, "", false
	}
	role, status := s.groupMemberRole(groupID, user.ID)
	if status != "active" {
		writeError(w, http.StatusForbidden, "You are not an active member of this group.")
		return CommunityUser{}, "", false
	}
	return user, role, true
}

func (s *Server) groupOwnerOrAdmin(w http.ResponseWriter, r *http.Request, groupID string) (CommunityUser, bool) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return CommunityUser{}, false
	}
	role, status := s.groupMemberRole(groupID, user.ID)
	if status == "active" && role == "owner" {
		return user, true
	}
	if s.canModerateCommunity(r, user) {
		return user, true
	}
	writeError(w, http.StatusForbidden, "Group owner permission required.")
	return CommunityUser{}, false
}

func (s *Server) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireCommunityWriter(w, r)
	if !ok {
		return
	}
	var req groupCreateRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	name, err := cleanCommunityText(req.Name, 80)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Group name is required.")
		return
	}
	intro := strings.TrimSpace(req.Intro)
	if len([]rune(intro)) > 500 {
		writeError(w, http.StatusBadRequest, "Group intro is too long.")
		return
	}
	visibility := normalizeGroupVisibility(req.Visibility)
	id := randomHex(8)
	now := nowISO()
	tx, err := s.db.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error.")
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO chat_groups(id, owner_id, name, avatar_url, intro, visibility, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?)`, id, user.ID, name, strings.TrimSpace(req.AvatarURL), intro, visibility, now, now); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create group.")
		return
	}
	if _, err := tx.Exec(`INSERT INTO chat_group_members(group_id, user_id, role, status, joined_at) VALUES(?,?, 'owner', 'active', ?)`, id, user.ID, now); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create group owner.")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create group.")
		return
	}
	s.handleGroup(w, r, id)
}

func normalizeGroupVisibility(v string) string {
	if strings.ToLower(strings.TrimSpace(v)) == "private" {
		return "private"
	}
	return "public"
}

func (s *Server) loadGroup(groupID, viewerPK string) (map[string]any, error) {
	row := s.db.QueryRow(`SELECT id, owner_id, name, COALESCE(avatar_url,''), COALESCE(intro,''), visibility, created_at, COALESCE(updated_at,''),
		(SELECT COUNT(*) FROM chat_group_members gm WHERE gm.group_id=chat_groups.id AND gm.status='active') AS member_count
		FROM chat_groups WHERE id=?`, groupID)
	return s.scanGroupRow(row, viewerPK)
}

func (s *Server) handleGroup(w http.ResponseWriter, r *http.Request, groupID string) {
	viewer, _, _ := s.communityUserFromRequest(r)
	group, err := s.loadGroup(groupID, viewer.ID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Group not found.")
		return
	}
	if group["visibility"] == "private" && group["my_status"] != "active" && !s.canModerateCommunity(r, viewer) {
		writeError(w, http.StatusForbidden, "This group is private.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"group": group})
}

func (s *Server) handleUpdateGroup(w http.ResponseWriter, r *http.Request, groupID string) {
	_, ok := s.groupOwnerOrAdmin(w, r, groupID)
	if !ok {
		return
	}
	var req groupCreateRequest
	if decodeJSON(r, &req) != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	name, err := cleanCommunityText(req.Name, 80)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Group name is required.")
		return
	}
	intro := strings.TrimSpace(req.Intro)
	if len([]rune(intro)) > 500 {
		writeError(w, http.StatusBadRequest, "Group intro is too long.")
		return
	}
	res, err := s.db.Exec(`UPDATE chat_groups SET name=?, avatar_url=?, intro=?, visibility=?, updated_at=? WHERE id=?`, name, strings.TrimSpace(req.AvatarURL), intro, normalizeGroupVisibility(req.Visibility), nowISO(), groupID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update group.")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeError(w, http.StatusNotFound, "Group not found.")
		return
	}
	s.handleGroup(w, r, groupID)
}

func (s *Server) handleDeleteGroup(w http.ResponseWriter, r *http.Request, groupID string) {
	_, ok := s.groupOwnerOrAdmin(w, r, groupID)
	if !ok {
		return
	}
	res, err := s.db.Exec(`DELETE FROM chat_groups WHERE id=?`, groupID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete group.")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeError(w, http.StatusNotFound, "Group not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleGroupMembers(w http.ResponseWriter, r *http.Request, groupID string) {
	if _, _, ok := s.requireActiveGroupMember(w, r, groupID); !ok {
		return
	}
	rows, err := s.db.Query(`SELECT u.id, u.user_id, u.nickname, COALESCE(u.avatar_url,''), COALESCE(u.bio,''), u.role, u.status, u.created_at, COALESCE(u.updated_at,''), COALESCE(u.last_login_at,''), gm.role, gm.status, gm.joined_at
		FROM chat_group_members gm JOIN community_users u ON u.id=gm.user_id
		WHERE gm.group_id=? AND gm.status='active' ORDER BY gm.joined_at ASC`, groupID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load members.")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var u CommunityUser
		var memberRole, memberStatus, joinedAt string
		if err := rows.Scan(&u.ID, &u.UserID, &u.Nickname, &u.AvatarURL, &u.Bio, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt, &memberRole, &memberStatus, &joinedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to read members.")
			return
		}
		m := publicCommunityUser(u)
		m["member_role"] = memberRole
		m["member_status"] = memberStatus
		m["joined_at"] = joinedAt
		items = append(items, m)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleJoinGroup(w http.ResponseWriter, r *http.Request, groupID string) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	group, err := s.loadGroup(groupID, user.ID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Group not found.")
		return
	}
	var req groupMemberRequest
	_ = decodeJSON(r, &req)
	targetID := user.ID
	if strings.TrimSpace(req.UserID) != "" {
		role, status := s.groupMemberRole(groupID, user.ID)
		if !(status == "active" && role == "owner") && !s.canModerateCommunity(r, user) {
			writeError(w, http.StatusForbidden, "Only group owner can add another user.")
			return
		}
		resolved, err := s.resolveCommunityUserPK(req.UserID)
		if err != nil {
			writeError(w, http.StatusNotFound, "Target user not found.")
			return
		}
		targetID = resolved
	}
	if group["visibility"] == "private" && targetID == user.ID && group["my_status"] != "active" {
		writeError(w, http.StatusForbidden, "This group is private.")
		return
	}
	var oldStatus string
	_ = s.db.QueryRow(`SELECT status FROM chat_group_members WHERE group_id=? AND user_id=?`, groupID, targetID).Scan(&oldStatus)
	if oldStatus == "kicked" && targetID == user.ID {
		writeError(w, http.StatusForbidden, "You were removed from this group.")
		return
	}
	now := nowISO()
	_, err = s.db.Exec(`INSERT INTO chat_group_members(group_id, user_id, role, status, joined_at) VALUES(?,?, 'member', 'active', ?)
		ON CONFLICT(group_id,user_id) DO UPDATE SET status='active', joined_at=excluded.joined_at`, groupID, targetID, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to join group.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "group_id": groupID, "user_pk": targetID})
}

func (s *Server) handleRemoveGroupMember(w http.ResponseWriter, r *http.Request, groupID, targetRaw string) {
	actor, ok := s.groupOwnerOrAdmin(w, r, groupID)
	if !ok {
		return
	}
	targetID, err := s.resolveCommunityUserPK(targetRaw)
	if err != nil {
		writeError(w, http.StatusNotFound, "Target user not found.")
		return
	}
	role, status := s.groupMemberRole(groupID, targetID)
	if status != "active" {
		writeError(w, http.StatusNotFound, "Member not found.")
		return
	}
	if role == "owner" && !s.canModerateCommunity(r, actor) {
		writeError(w, http.StatusForbidden, "Cannot remove group owner.")
		return
	}
	_, err = s.db.Exec(`UPDATE chat_group_members SET status='kicked' WHERE group_id=? AND user_id=?`, groupID, targetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to remove member.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleGroupMessages(w http.ResponseWriter, r *http.Request, groupID string) {
	if _, _, ok := s.requireActiveGroupMember(w, r, groupID); !ok {
		return
	}
	limit := parseLimit(r, 50, 100)
	rows, err := s.db.Query(`SELECT m.id, m.sender_id, m.content, m.created_at, u.user_id, u.nickname, COALESCE(u.avatar_url,'')
		FROM chat_group_messages m JOIN community_users u ON u.id=m.sender_id
		WHERE m.group_id=? AND m.status='normal' ORDER BY m.created_at DESC LIMIT ?`, groupID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load group messages.")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, senderPK, content, createdAt, senderID, nickname, avatar string
		if err := rows.Scan(&id, &senderPK, &content, &createdAt, &senderID, &nickname, &avatar); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to read group messages.")
			return
		}
		items = append(items, map[string]any{"id": id, "content": content, "created_at": createdAt, "sender": map[string]any{"id": senderID, "user_pk": senderPK, "nickname": nickname, "avatar_url": avatar}})
	}
	reverseMaps(items)
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleSendGroupMessage(w http.ResponseWriter, r *http.Request, groupID string) {
	user, _, ok := s.requireActiveGroupMember(w, r, groupID)
	if !ok {
		return
	}
	if user.Status == "muted" {
		writeError(w, http.StatusForbidden, "This user is muted.")
		return
	}
	if !s.checkRateLimit("gm:"+user.ID+":"+clientIP(r), 80, 10*time.Minute, true) {
		writeError(w, http.StatusTooManyRequests, "Group message requests are too frequent.")
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
	if _, err := s.db.Exec(`INSERT INTO chat_group_messages(id, group_id, sender_id, content, status, created_at) VALUES(?,?,?,?,'normal',?)`, id, groupID, user.ID, content, now); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to send group message.")
		return
	}
	s.notifyGroupMembers(groupID, user.ID, "group_message", map[string]any{"group_id": groupID, "message_id": id, "actor_id": user.UserID, "actor_nickname": user.Nickname, "preview": previewText(content, 80)})
	writeJSON(w, http.StatusOK, map[string]any{"message": map[string]any{"id": id, "group_id": groupID, "content": content, "created_at": now, "sender": publicCommunityUser(user)}})
}
