package app

import (
	"database/sql"
	"net/http"
	"strings"
)

type AdminCommunityUserCreateRequest struct {
	UserID    string `json:"user_id"`
	Nickname  string `json:"nickname"`
	Password  string `json:"password"`
	AvatarURL string `json:"avatar_url"`
	Bio       string `json:"bio"`
	Role      string `json:"role"`
	Status    string `json:"status"`
}

type AdminCommunityUserUpdateRequest struct {
	UserID    string `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	Bio       string `json:"bio"`
	Role      string `json:"role"`
	Status    string `json:"status"`
}

type AdminCommunityStatusUpdateRequest struct {
	Status string `json:"status"`
}

type AdminCommunityRoleUpdateRequest struct {
	Role string `json:"role"`
}

type AdminCommunityPasswordUpdateRequest struct {
	Password string `json:"password"`
}

func (s *Server) routeAdminBlogOps(w http.ResponseWriter, r *http.Request, path string) bool {
	switch {
	case path == "/api/admin/blog/site-state":
		switch r.Method {
		case http.MethodGet:
			s.handleGetBlogSiteState(w, r)
		case http.MethodPut:
			s.handleUpdateBlogSiteState(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		}
		return true
	case path == "/api/admin/community/users":
		switch r.Method {
		case http.MethodGet:
			s.handleAdminCommunityUsers(w, r)
		case http.MethodPost:
			s.handleCreateAdminCommunityUser(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		}
		return true
	case strings.HasPrefix(path, "/api/admin/community/users/") && strings.HasSuffix(path, "/status"):
		if r.Method != http.MethodPut {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
			return true
		}
		s.handleUpdateAdminCommunityUserStatus(w, r, strings.TrimSuffix(pathValue(path, "/api/admin/community/users/"), "/status"))
		return true
	case strings.HasPrefix(path, "/api/admin/community/users/") && strings.HasSuffix(path, "/role"):
		if r.Method != http.MethodPut {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
			return true
		}
		s.handleUpdateAdminCommunityUserRole(w, r, strings.TrimSuffix(pathValue(path, "/api/admin/community/users/"), "/role"))
		return true
	case strings.HasPrefix(path, "/api/admin/community/users/") && strings.HasSuffix(path, "/password"):
		if r.Method != http.MethodPut {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
			return true
		}
		s.handleResetAdminCommunityUserPassword(w, r, strings.TrimSuffix(pathValue(path, "/api/admin/community/users/"), "/password"))
		return true
	case strings.HasPrefix(path, "/api/admin/community/users/"):
		id := pathValue(path, "/api/admin/community/users/")
		switch r.Method {
		case http.MethodPut:
			s.handleUpdateAdminCommunityUser(w, r, id)
		case http.MethodDelete:
			s.handleDeleteAdminCommunityUser(w, r, id)
		default:
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		}
		return true
	case path == "/api/superadmin/audit/private-conversations" && r.Method == http.MethodGet:
		s.handleSuperAuditPrivateConversations(w, r)
		return true
	case strings.HasPrefix(path, "/api/superadmin/audit/private-conversations/") && strings.HasSuffix(path, "/messages") && r.Method == http.MethodGet:
		s.handleSuperAuditPrivateMessages(w, r, strings.TrimSuffix(pathValue(path, "/api/superadmin/audit/private-conversations/"), "/messages"))
		return true
	case path == "/api/superadmin/audit/groups" && r.Method == http.MethodGet:
		s.handleSuperAuditGroups(w, r)
		return true
	case strings.HasPrefix(path, "/api/superadmin/audit/groups/") && strings.HasSuffix(path, "/messages") && r.Method == http.MethodGet:
		s.handleSuperAuditGroupMessages(w, r, strings.TrimSuffix(pathValue(path, "/api/superadmin/audit/groups/"), "/messages"))
		return true
	case strings.HasPrefix(path, "/api/superadmin/audit/"):
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		return true
	default:
		return false
	}
}

func parseCommunityStatus(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "active":
		return "active", true
	case "muted":
		return "muted", true
	case "banned":
		return "banned", true
	case "pending":
		return "pending", true
	case "rejected":
		return "rejected", true
	case "deleted":
		return "deleted", true
	default:
		return "", false
	}
}

func parseCommunityRoleForAdmin(raw string, superAdmin bool) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "user":
		return "user", true
	case "moderator":
		return "moderator", true
	case "admin", "superadmin":
		if superAdmin {
			return strings.ToLower(strings.TrimSpace(raw)), true
		}
		return "", false
	default:
		return "", false
	}
}

func adminCommunityRoleLabel(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "superadmin":
		return "超级管理员"
	case "admin":
		return "管理员"
	case "moderator":
		return "社区协管"
	default:
		return "普通用户"
	}
}

func adminCommunityStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "muted":
		return "\u7981\u8a00"
	case "banned":
		return "\u5c01\u7981"
	case "pending":
		return "\u5f85\u5ba1\u6838"
	case "rejected":
		return "\u5df2\u9a73\u56de"
	case "deleted":
		return "\u5df2\u5220\u9664"
	default:
		return "\u6b63\u5e38"
	}
}

func (s *Server) canAdminManageCommunityUser(admin AdminSession, target CommunityUser) bool {
	if normalizeRole(admin.Role) == "superadmin" {
		return true
	}
	role := strings.ToLower(strings.TrimSpace(target.Role))
	return role == "" || role == "user" || role == "moderator"
}

func (s *Server) resolveCommunityUserPKAny(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || s.db == nil {
		return "", sql.ErrNoRows
	}
	var id string
	err := s.db.QueryRow(`SELECT id FROM community_users WHERE id=? OR user_id=?`, raw, raw).Scan(&id)
	return id, err
}

func (s *Server) loadCommunityUserByPKAny(id string) (CommunityUser, error) {
	var u CommunityUser
	err := s.db.QueryRow(`SELECT id, user_id, nickname, COALESCE(avatar_url,''), COALESCE(bio,''), role, status, created_at, COALESCE(updated_at,''), COALESCE(last_login_at,'')
		FROM community_users WHERE id=?`, id).Scan(&u.ID, &u.UserID, &u.Nickname, &u.AvatarURL, &u.Bio, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt)
	return u, err
}

func (s *Server) dropCommunitySessionsFor(userPK string) {
	if s.db == nil || strings.TrimSpace(userPK) == "" {
		return
	}
	_, _ = s.db.Exec(`DELETE FROM community_sessions WHERE user_pk=?`, userPK)
}

func (s *Server) publicAdminCommunityUser(u CommunityUser) map[string]any {
	out := publicCommunityUser(u)
	out["role_label"] = adminCommunityRoleLabel(u.Role)
	out["status_label"] = adminCommunityStatusLabel(u.Status)
	var articles, comments, followers, following, privateMessages, groupMessages int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM blog_articles WHERE author_id=?`, u.ID).Scan(&articles)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM blog_comments WHERE author_id=?`, u.ID).Scan(&comments)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM social_follows WHERE following_id=?`, u.ID).Scan(&followers)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM social_follows WHERE follower_id=?`, u.ID).Scan(&following)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM private_messages WHERE sender_id=?`, u.ID).Scan(&privateMessages)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM chat_group_messages WHERE sender_id=?`, u.ID).Scan(&groupMessages)
	out["stats"] = map[string]any{
		"articles":         articles,
		"comments":         comments,
		"followers":        followers,
		"following":        following,
		"private_messages": privateMessages,
		"group_messages":   groupMessages,
	}
	return out
}

func (s *Server) handleAdminCommunityUsers(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireBlogAdmin(w, r)
	if !ok {
		return
	}
	if s.db == nil {
		writeError(w, http.StatusServiceUnavailable, "Database unavailable.")
		return
	}
	page, pageSize, offset := parsePage(r, 40, 100)
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	includeDeleted := normalizeRole(admin.Role) == "superadmin" && (r.URL.Query().Get("include_deleted") == "1" || strings.EqualFold(r.URL.Query().Get("include_deleted"), "true"))

	where := []string{"1=1"}
	args := []any{}
	if !includeDeleted {
		where = append(where, "status!='deleted'")
	}
	if normalizeRole(admin.Role) != "superadmin" {
		where = append(where, "LOWER(role) NOT IN ('admin','superadmin')")
	}
	if query != "" {
		like := "%" + query + "%"
		where = append(where, "(LOWER(user_id) LIKE ? OR LOWER(nickname) LIKE ? OR LOWER(COALESCE(bio,'')) LIKE ?)")
		args = append(args, like, like, like)
	}
	whereSQL := strings.Join(where, " AND ")
	var total int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM community_users WHERE `+whereSQL, args...).Scan(&total)
	pendingWhere := []string{"status='pending'"}
	if normalizeRole(admin.Role) != "superadmin" {
		pendingWhere = append(pendingWhere, "LOWER(role) NOT IN ('admin','superadmin')")
	}
	var pendingCount int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM community_users WHERE ` + strings.Join(pendingWhere, " AND ")).Scan(&pendingCount)
	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, pageSize, offset)
	rows, err := s.db.Query(`SELECT id, user_id, nickname, COALESCE(avatar_url,''), COALESCE(bio,''), role, status, created_at, COALESCE(updated_at,''), COALESCE(last_login_at,'')
		FROM community_users WHERE `+whereSQL+`
		ORDER BY CASE status WHEN 'pending' THEN 0 WHEN 'active' THEN 1 WHEN 'muted' THEN 2 WHEN 'banned' THEN 3 ELSE 4 END,
		CASE LOWER(role) WHEN 'superadmin' THEN 0 WHEN 'admin' THEN 1 WHEN 'moderator' THEN 2 ELSE 3 END,
		created_at DESC LIMIT ? OFFSET ?`, listArgs...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load community users.")
		return
	}
	rawUsers := []CommunityUser{}
	for rows.Next() {
		var u CommunityUser
		if err := rows.Scan(&u.ID, &u.UserID, &u.Nickname, &u.AvatarURL, &u.Bio, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt); err != nil {
			_ = rows.Close()
			writeError(w, http.StatusInternalServerError, "Failed to read community users.")
			return
		}
		rawUsers = append(rawUsers, u)
	}
	if err := rows.Close(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read community users.")
		return
	}
	items := []map[string]any{}
	for _, u := range rawUsers {
		items = append(items, s.publicAdminCommunityUser(u))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "pending_count": pendingCount, "page": page, "page_size": pageSize, "has_more": offset+len(items) < total})
}

func (s *Server) handleCreateAdminCommunityUser(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireBlogAdmin(w, r)
	if !ok {
		return
	}
	var req AdminCommunityUserCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	userID, err := validateCommunityUserID(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	nickname, err := validateCommunityNickname(req.Nickname)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateCommunityPassword(req.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	role, okRole := parseCommunityRoleForAdmin(req.Role, normalizeRole(admin.Role) == "superadmin")
	if !okRole {
		writeError(w, http.StatusForbidden, "Only super administrators can grant privileged community roles.")
		return
	}
	status, okStatus := parseCommunityStatus(req.Status)
	if !okStatus || status == "deleted" || status == "rejected" {
		writeError(w, http.StatusBadRequest, "Invalid user status.")
		return
	}
	bio := strings.TrimSpace(req.Bio)
	if len([]rune(bio)) > 500 {
		writeError(w, http.StatusBadRequest, "Bio must be 500 characters or fewer.")
		return
	}
	avatarURL := strings.TrimSpace(req.AvatarURL)
	if len([]rune(avatarURL)) > 500 {
		writeError(w, http.StatusBadRequest, "Avatar URL is too long.")
		return
	}
	var exists int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM community_users WHERE user_id=?`, userID).Scan(&exists)
	if exists > 0 {
		writeError(w, http.StatusConflict, "User ID already exists.")
		return
	}
	salt, hash := hashPassword(req.Password, "")
	u := CommunityUser{ID: randomHex(6), UserID: userID, Nickname: nickname, AvatarURL: avatarURL, Bio: bio, Role: role, Status: status, CreatedAt: nowISO()}
	_, err = s.db.Exec(`INSERT INTO community_users(id, user_id, nickname, password_hash, salt, avatar_url, bio, role, status, created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, u.ID, u.UserID, u.Nickname, hash, salt, u.AvatarURL, u.Bio, u.Role, u.Status, u.CreatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create community user.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": s.publicAdminCommunityUser(u)})
}

func (s *Server) handleUpdateAdminCommunityUser(w http.ResponseWriter, r *http.Request, raw string) {
	admin, ok := s.requireBlogAdmin(w, r)
	if !ok {
		return
	}
	pk, err := s.resolveCommunityUserPKAny(raw)
	if err != nil {
		writeError(w, http.StatusNotFound, "User not found.")
		return
	}
	target, err := s.loadCommunityUserByPKAny(pk)
	if err != nil {
		writeError(w, http.StatusNotFound, "User not found.")
		return
	}
	if !s.canAdminManageCommunityUser(admin, target) {
		writeError(w, http.StatusForbidden, "Only super administrators can manage privileged users.")
		return
	}
	var req AdminCommunityUserUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	userID := target.UserID
	if strings.TrimSpace(req.UserID) != "" {
		userID, err = validateCommunityUserID(req.UserID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		var exists int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM community_users WHERE user_id=? AND id!=?`, userID, pk).Scan(&exists)
		if exists > 0 {
			writeError(w, http.StatusConflict, "User ID already exists.")
			return
		}
	}
	nickname := target.Nickname
	if strings.TrimSpace(req.Nickname) != "" {
		nickname, err = validateCommunityNickname(req.Nickname)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	bio := target.Bio
	if req.Bio != "" {
		bio = strings.TrimSpace(req.Bio)
	}
	if len([]rune(bio)) > 500 {
		writeError(w, http.StatusBadRequest, "Bio must be 500 characters or fewer.")
		return
	}
	avatarURL := target.AvatarURL
	if req.AvatarURL != "" {
		avatarURL = strings.TrimSpace(req.AvatarURL)
	}
	if len([]rune(avatarURL)) > 500 {
		writeError(w, http.StatusBadRequest, "Avatar URL is too long.")
		return
	}
	role := target.Role
	if strings.TrimSpace(req.Role) != "" {
		var roleOK bool
		role, roleOK = parseCommunityRoleForAdmin(req.Role, normalizeRole(admin.Role) == "superadmin")
		if !roleOK {
			writeError(w, http.StatusForbidden, "Only super administrators can grant privileged community roles.")
			return
		}
	}
	status := target.Status
	if strings.TrimSpace(req.Status) != "" {
		var statusOK bool
		status, statusOK = parseCommunityStatus(req.Status)
		if !statusOK {
			writeError(w, http.StatusBadRequest, "Invalid user status.")
			return
		}
	}
	_, err = s.db.Exec(`UPDATE community_users SET user_id=?, nickname=?, avatar_url=?, bio=?, role=?, status=?, updated_at=? WHERE id=?`, userID, nickname, avatarURL, bio, role, status, nowISO(), pk)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update user.")
		return
	}
	if status == "banned" || status == "deleted" {
		s.dropCommunitySessionsFor(pk)
	}
	u, _ := s.loadCommunityUserByPKAny(pk)
	writeJSON(w, http.StatusOK, map[string]any{"user": s.publicAdminCommunityUser(u)})
}

func (s *Server) handleUpdateAdminCommunityUserStatus(w http.ResponseWriter, r *http.Request, raw string) {
	admin, ok := s.requireBlogAdmin(w, r)
	if !ok {
		return
	}
	pk, err := s.resolveCommunityUserPKAny(raw)
	if err != nil {
		writeError(w, http.StatusNotFound, "User not found.")
		return
	}
	target, err := s.loadCommunityUserByPKAny(pk)
	if err != nil {
		writeError(w, http.StatusNotFound, "User not found.")
		return
	}
	if !s.canAdminManageCommunityUser(admin, target) {
		writeError(w, http.StatusForbidden, "Only super administrators can manage privileged users.")
		return
	}
	var req AdminCommunityStatusUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	status, okStatus := parseCommunityStatus(req.Status)
	if !okStatus || status == "deleted" {
		writeError(w, http.StatusBadRequest, "Invalid user status.")
		return
	}
	currentStatus := strings.ToLower(strings.TrimSpace(target.Status))
	if status == "rejected" {
		if currentStatus != "pending" {
			writeError(w, http.StatusBadRequest, "Only pending registrations can be rejected.")
			return
		}
		_, err = s.db.Exec(`DELETE FROM community_users WHERE id=? AND status='pending'`, pk)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to reject registration.")
			return
		}
		s.dropCommunitySessionsFor(pk)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "rejected": pk, "released_user_id": target.UserID})
		return
	}
	if currentStatus == "pending" && status != "active" {
		writeError(w, http.StatusBadRequest, "Pending registrations can only be approved or rejected.")
		return
	}
	if currentStatus != "pending" && status == "pending" {
		writeError(w, http.StatusBadRequest, "Active users cannot be moved back to pending review.")
		return
	}
	_, err = s.db.Exec(`UPDATE community_users SET status=?, updated_at=? WHERE id=?`, status, nowISO(), pk)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update user status.")
		return
	}
	if status == "banned" {
		s.dropCommunitySessionsFor(pk)
	}
	u, _ := s.loadCommunityUserByPKAny(pk)
	writeJSON(w, http.StatusOK, map[string]any{"user": s.publicAdminCommunityUser(u)})
}

func (s *Server) handleUpdateAdminCommunityUserRole(w http.ResponseWriter, r *http.Request, raw string) {
	admin, ok := s.requireBlogAdmin(w, r)
	if !ok {
		return
	}
	pk, err := s.resolveCommunityUserPKAny(raw)
	if err != nil {
		writeError(w, http.StatusNotFound, "User not found.")
		return
	}
	target, err := s.loadCommunityUserByPKAny(pk)
	if err != nil {
		writeError(w, http.StatusNotFound, "User not found.")
		return
	}
	if !s.canAdminManageCommunityUser(admin, target) {
		writeError(w, http.StatusForbidden, "Only super administrators can manage privileged users.")
		return
	}
	var req AdminCommunityRoleUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	role, okRole := parseCommunityRoleForAdmin(req.Role, normalizeRole(admin.Role) == "superadmin")
	if !okRole {
		writeError(w, http.StatusForbidden, "Only super administrators can grant privileged community roles.")
		return
	}
	_, err = s.db.Exec(`UPDATE community_users SET role=?, updated_at=? WHERE id=?`, role, nowISO(), pk)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update user role.")
		return
	}
	u, _ := s.loadCommunityUserByPKAny(pk)
	writeJSON(w, http.StatusOK, map[string]any{"user": s.publicAdminCommunityUser(u)})
}

func (s *Server) handleResetAdminCommunityUserPassword(w http.ResponseWriter, r *http.Request, raw string) {
	admin, ok := s.requireBlogAdmin(w, r)
	if !ok {
		return
	}
	pk, err := s.resolveCommunityUserPKAny(raw)
	if err != nil {
		writeError(w, http.StatusNotFound, "User not found.")
		return
	}
	target, err := s.loadCommunityUserByPKAny(pk)
	if err != nil {
		writeError(w, http.StatusNotFound, "User not found.")
		return
	}
	if !s.canAdminManageCommunityUser(admin, target) {
		writeError(w, http.StatusForbidden, "Only super administrators can manage privileged users.")
		return
	}
	var req AdminCommunityPasswordUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	if err := validateCommunityPassword(req.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	salt, hash := hashPassword(req.Password, "")
	_, err = s.db.Exec(`UPDATE community_users SET salt=?, password_hash=?, updated_at=? WHERE id=?`, salt, hash, nowISO(), pk)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to reset password.")
		return
	}
	s.dropCommunitySessionsFor(pk)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleDeleteAdminCommunityUser(w http.ResponseWriter, r *http.Request, raw string) {
	admin, ok := s.requireBlogAdmin(w, r)
	if !ok {
		return
	}
	pk, err := s.resolveCommunityUserPKAny(raw)
	if err != nil {
		writeError(w, http.StatusNotFound, "User not found.")
		return
	}
	target, err := s.loadCommunityUserByPKAny(pk)
	if err != nil {
		writeError(w, http.StatusNotFound, "User not found.")
		return
	}
	if !s.canAdminManageCommunityUser(admin, target) {
		writeError(w, http.StatusForbidden, "Only super administrators can manage privileged users.")
		return
	}
	_, err = s.db.Exec(`UPDATE community_users SET status='deleted', updated_at=? WHERE id=?`, nowISO(), pk)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete user.")
		return
	}
	s.dropCommunitySessionsFor(pk)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted": pk})
}

func (s *Server) auditCommunityUserPayload(pk string) map[string]any {
	u, err := s.loadCommunityUserByPKAny(pk)
	if err != nil {
		return map[string]any{"user_pk": pk, "id": pk, "user_id": pk, "nickname": "已删除用户", "avatar_url": "", "status": "deleted"}
	}
	return publicCommunityUser(u)
}

func (s *Server) handleSuperAuditPrivateConversations(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	page, pageSize, offset := parsePage(r, 30, 100)
	rows, err := s.db.Query(`SELECT c.id, c.user_a, c.user_b, c.created_at, COALESCE(c.updated_at,''), COALESCE(c.last_message_at,''),
		(SELECT COUNT(*) FROM private_messages pm WHERE pm.conversation_id=c.id) AS message_count,
		COALESCE((SELECT pm.content FROM private_messages pm WHERE pm.conversation_id=c.id ORDER BY pm.created_at DESC LIMIT 1),''),
		COALESCE((SELECT pm.sender_id FROM private_messages pm WHERE pm.conversation_id=c.id ORDER BY pm.created_at DESC LIMIT 1),''),
		COALESCE((SELECT pm.created_at FROM private_messages pm WHERE pm.conversation_id=c.id ORDER BY pm.created_at DESC LIMIT 1),'')
		FROM private_conversations c
		ORDER BY COALESCE(c.last_message_at,c.created_at) DESC LIMIT ? OFFSET ?`, pageSize, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load private conversations.")
		return
	}
	type auditPrivateConversationRow struct {
		id, userA, userB, createdAt, updatedAt, lastAt, lastContent, lastSender, lastCreated string
		messageCount                                                                         int
	}
	rawItems := []auditPrivateConversationRow{}
	for rows.Next() {
		var item auditPrivateConversationRow
		if err := rows.Scan(&item.id, &item.userA, &item.userB, &item.createdAt, &item.updatedAt, &item.lastAt, &item.messageCount, &item.lastContent, &item.lastSender, &item.lastCreated); err != nil {
			_ = rows.Close()
			writeError(w, http.StatusInternalServerError, "Failed to read private conversations.")
			return
		}
		rawItems = append(rawItems, item)
	}
	if err := rows.Close(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read private conversations.")
		return
	}
	items := []map[string]any{}
	for _, item := range rawItems {
		items = append(items, map[string]any{
			"id":              item.id,
			"created_at":      item.createdAt,
			"updated_at":      item.updatedAt,
			"last_message_at": item.lastAt,
			"message_count":   item.messageCount,
			"participants":    []map[string]any{s.auditCommunityUserPayload(item.userA), s.auditCommunityUserPayload(item.userB)},
			"last_message":    map[string]any{"content": item.lastContent, "sender": s.auditCommunityUserPayload(item.lastSender), "created_at": item.lastCreated},
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "page": page, "page_size": pageSize, "has_more": len(items) == pageSize})
}

func (s *Server) handleSuperAuditPrivateMessages(w http.ResponseWriter, r *http.Request, conversationID string) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		writeError(w, http.StatusBadRequest, "Conversation id is required.")
		return
	}
	limit := parseLimit(r, 100, 300)
	rows, err := s.db.Query(`SELECT m.id, m.sender_id, m.content, m.status, m.created_at, COALESCE(m.read_at,''), COALESCE(u.user_id,''), COALESCE(u.nickname,''), COALESCE(u.avatar_url,''), COALESCE(u.status,'deleted')
		FROM private_messages m LEFT JOIN community_users u ON u.id=m.sender_id
		WHERE m.conversation_id=? ORDER BY m.created_at DESC LIMIT ?`, conversationID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load private messages.")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, senderPK, content, status, createdAt, readAt, userID, nickname, avatarURL, userStatus string
		if err := rows.Scan(&id, &senderPK, &content, &status, &createdAt, &readAt, &userID, &nickname, &avatarURL, &userStatus); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to read private messages.")
			return
		}
		if userID == "" {
			userID = senderPK
		}
		if nickname == "" {
			nickname = "已删除用户"
		}
		items = append(items, map[string]any{"id": id, "conversation_id": conversationID, "content": content, "status": status, "created_at": createdAt, "read_at": readAt, "sender": map[string]any{"id": userID, "user_id": userID, "user_pk": senderPK, "nickname": nickname, "avatar_url": avatarURL, "status": userStatus}})
	}
	reverseMaps(items)
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "conversation_id": conversationID})
}

func (s *Server) handleSuperAuditGroups(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	page, pageSize, offset := parsePage(r, 30, 100)
	rows, err := s.db.Query(`SELECT g.id, g.owner_id, g.name, COALESCE(g.avatar_url,''), COALESCE(g.intro,''), g.visibility, g.created_at, COALESCE(g.updated_at,''),
		(SELECT COUNT(*) FROM chat_group_members gm WHERE gm.group_id=g.id AND gm.status='active') AS member_count,
		(SELECT COUNT(*) FROM chat_group_messages msg WHERE msg.group_id=g.id) AS message_count,
		COALESCE((SELECT msg.content FROM chat_group_messages msg WHERE msg.group_id=g.id ORDER BY msg.created_at DESC LIMIT 1),''),
		COALESCE((SELECT msg.sender_id FROM chat_group_messages msg WHERE msg.group_id=g.id ORDER BY msg.created_at DESC LIMIT 1),''),
		COALESCE((SELECT msg.created_at FROM chat_group_messages msg WHERE msg.group_id=g.id ORDER BY msg.created_at DESC LIMIT 1),'')
		FROM chat_groups g ORDER BY COALESCE((SELECT msg.created_at FROM chat_group_messages msg WHERE msg.group_id=g.id ORDER BY msg.created_at DESC LIMIT 1), g.created_at) DESC LIMIT ? OFFSET ?`, pageSize, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load groups.")
		return
	}
	type auditGroupRow struct {
		id, ownerID, name, avatar, intro, visibility, createdAt, updatedAt, lastContent, lastSender, lastCreated string
		memberCount, messageCount                                                                                int
	}
	rawItems := []auditGroupRow{}
	for rows.Next() {
		var item auditGroupRow
		if err := rows.Scan(&item.id, &item.ownerID, &item.name, &item.avatar, &item.intro, &item.visibility, &item.createdAt, &item.updatedAt, &item.memberCount, &item.messageCount, &item.lastContent, &item.lastSender, &item.lastCreated); err != nil {
			_ = rows.Close()
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
	for _, item := range rawItems {
		items = append(items, map[string]any{"id": item.id, "name": item.name, "avatar_url": item.avatar, "intro": item.intro, "visibility": item.visibility, "created_at": item.createdAt, "updated_at": item.updatedAt, "owner": s.auditCommunityUserPayload(item.ownerID), "member_count": item.memberCount, "message_count": item.messageCount, "last_message": map[string]any{"content": item.lastContent, "sender": s.auditCommunityUserPayload(item.lastSender), "created_at": item.lastCreated}})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "page": page, "page_size": pageSize, "has_more": len(items) == pageSize})
}

func (s *Server) handleSuperAuditGroupMessages(w http.ResponseWriter, r *http.Request, groupID string) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		writeError(w, http.StatusBadRequest, "Group id is required.")
		return
	}
	limit := parseLimit(r, 100, 300)
	rows, err := s.db.Query(`SELECT m.id, m.sender_id, m.content, m.status, m.created_at, COALESCE(u.user_id,''), COALESCE(u.nickname,''), COALESCE(u.avatar_url,''), COALESCE(u.status,'deleted')
		FROM chat_group_messages m LEFT JOIN community_users u ON u.id=m.sender_id
		WHERE m.group_id=? ORDER BY m.created_at DESC LIMIT ?`, groupID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load group messages.")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, senderPK, content, status, createdAt, userID, nickname, avatarURL, userStatus string
		if err := rows.Scan(&id, &senderPK, &content, &status, &createdAt, &userID, &nickname, &avatarURL, &userStatus); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to read group messages.")
			return
		}
		if userID == "" {
			userID = senderPK
		}
		if nickname == "" {
			nickname = "已删除用户"
		}
		items = append(items, map[string]any{"id": id, "group_id": groupID, "content": content, "status": status, "created_at": createdAt, "sender": map[string]any{"id": userID, "user_id": userID, "user_pk": senderPK, "nickname": nickname, "avatar_url": avatarURL, "status": userStatus}})
	}
	reverseMaps(items)
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "group_id": groupID})
}
