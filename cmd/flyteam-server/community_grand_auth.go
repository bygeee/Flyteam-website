package main

import (
	"database/sql"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type CommunityRegisterRequest struct {
	Nickname string `json:"nickname"`
	UserID   string `json:"user_id"`
	Password string `json:"password"`
}

type CommunityLoginRequest struct {
	UserID   string `json:"user_id"`
	Password string `json:"password"`
}

type CommunityProfileUpdateRequest struct {
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	Bio       string `json:"bio"`
}

type CommunityAccountUpdateRequest struct {
	Nickname  string `json:"nickname"`
	UserID    string `json:"user_id"`
	AvatarURL string `json:"avatar_url"`
	Bio       string `json:"bio"`
}

type CommunityPasswordUpdateRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

var communityUserIDRe = regexp.MustCompile(`^[0-9A-Za-z_\-]{3,32}$`)
var reservedCommunityUserIDs = map[string]bool{"admin": true, "root": true, "api": true, "static": true, "uploads": true, "login": true, "blog": true, "editor": true, "account": true, "messages": true, "groups": true, "user-login": true, "user-register": true}

func validateCommunityUserID(userID string) (string, error) {
	clean := strings.ToLower(strings.TrimSpace(userID))
	if !communityUserIDRe.MatchString(clean) {
		return "", errors.New("User ID must be 3-32 chars: letters, numbers, _, -.")
	}
	if reservedCommunityUserIDs[clean] {
		return "", errors.New("This User ID is reserved.")
	}
	return clean, nil
}

func validateCommunityNickname(nickname string) (string, error) {
	clean := strings.TrimSpace(nickname)
	if len([]rune(clean)) < 1 || len([]rune(clean)) > 30 {
		return "", errors.New("Nickname must be 1-30 characters.")
	}
	return clean, nil
}

func validateCommunityPassword(password string) error {
	if len([]rune(password)) < 8 {
		return errors.New("Password must be at least 8 characters.")
	}
	low := strings.ToLower(password)
	if strings.Contains(low, "password") || strings.Contains(low, "123456") || strings.TrimSpace(password) != password {
		return errors.New("Password is too weak.")
	}
	return nil
}

func (s *Server) issueCommunitySession(u CommunityUser) (string, string, time.Time, error) {
	token := randomHex(32)
	csrf := randomHex(32)
	hours := s.cfg.UserSessionHours
	if hours < 1 {
		hours = 1
	}
	expires := time.Now().UTC().Add(time.Duration(hours) * time.Hour)
	if s.db != nil {
		_, err := s.db.Exec(`INSERT INTO community_sessions(session_token, user_pk, csrf_token, expires_at, created_at, user_agent_hash, ip_hash)
			VALUES(?,?,?,?,?,?,?)
			ON CONFLICT(session_token) DO UPDATE SET user_pk=excluded.user_pk, csrf_token=excluded.csrf_token, expires_at=excluded.expires_at`, token, u.ID, csrf, expires.Format(time.RFC3339Nano), nowISO(), nil, nil)
		if err != nil {
			return "", "", time.Time{}, err
		}
	}
	return token, csrf, expires, nil
}

func (s *Server) findCommunityLoginUser(userID string) (CommunityUser, string, string, error) {
	var u CommunityUser
	var passwordHash, salt string
	err := s.db.QueryRow(`SELECT id, user_id, nickname, password_hash, salt, COALESCE(avatar_url,''), COALESCE(bio,''), role, status, created_at, COALESCE(updated_at,''), COALESCE(last_login_at,'')
		FROM community_users WHERE user_id=? AND status!='deleted'`, userID).Scan(&u.ID, &u.UserID, &u.Nickname, &passwordHash, &salt, &u.AvatarURL, &u.Bio, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt)
	return u, passwordHash, salt, err
}

func (s *Server) handleCommunityRegister(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeError(w, http.StatusServiceUnavailable, "Database unavailable.")
		return
	}
	registerKey := "user-register:" + clientIP(r)
	if !s.checkRateLimit(registerKey, 6, time.Hour, false) {
		writeError(w, http.StatusTooManyRequests, "Too many registration attempts. Please try again later.")
		return
	}
	var req CommunityRegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		s.checkRateLimit(registerKey, 6, time.Hour, true)
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	nickname, err := validateCommunityNickname(req.Nickname)
	if err != nil {
		s.checkRateLimit(registerKey, 6, time.Hour, true)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	userID, err := validateCommunityUserID(req.UserID)
	if err != nil {
		s.checkRateLimit(registerKey, 6, time.Hour, true)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateCommunityPassword(req.Password); err != nil {
		s.checkRateLimit(registerKey, 6, time.Hour, true)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.checkRateLimit(registerKey, 6, time.Hour, true)
	var existingID, existingStatus string
	err = s.db.QueryRow(`SELECT id, status FROM community_users WHERE user_id=?`, userID).Scan(&existingID, &existingStatus)
	if err == nil {
		switch strings.ToLower(strings.TrimSpace(existingStatus)) {
		case "pending":
			writeError(w, http.StatusConflict, "\u8be5\u8d26\u53f7\u6ce8\u518c\u7533\u8bf7\u6b63\u5728\u7b49\u5f85\u7ba1\u7406\u5458\u5ba1\u6838\u3002")
			return
		case "rejected":
			_, _ = s.db.Exec(`DELETE FROM community_users WHERE id=? AND status='rejected'`, existingID)
		default:
			writeError(w, http.StatusConflict, "User ID already exists.")
			return
		}
	} else if err != nil && err != sql.ErrNoRows {
		writeError(w, http.StatusInternalServerError, "Failed to check user ID.")
		return
	}
	salt, hash := hashPassword(req.Password, "")
	u := CommunityUser{ID: randomHex(6), UserID: userID, Nickname: nickname, Role: "user", Status: "pending", CreatedAt: nowISO()}
	_, err = s.db.Exec(`INSERT INTO community_users(id, user_id, nickname, password_hash, salt, role, status, created_at) VALUES(?,?,?,?,?,?,?,?)`, u.ID, u.UserID, u.Nickname, hash, salt, u.Role, u.Status, u.CreatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save registration application.")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "status": "pending", "message": "\u6ce8\u518c\u7533\u8bf7\u5df2\u63d0\u4ea4\uff0c\u8bf7\u7b49\u5f85\u535a\u5ba2\u7ad9\u7ba1\u7406\u5458\u6216\u8d85\u7ea7\u7ba1\u7406\u5458\u5ba1\u6838\u901a\u8fc7\u540e\u518d\u767b\u5f55\u3002", "user": publicCommunityUser(u)})
}

func (s *Server) handleCommunityLogin(w http.ResponseWriter, r *http.Request) {
	var req CommunityLoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	userID, err := validateCommunityUserID(req.UserID)
	if err != nil || req.Password == "" {
		writeError(w, http.StatusBadRequest, "User ID and password are required.")
		return
	}
	loginKey := "user-login:" + clientIP(r) + ":" + userID
	if !s.checkRateLimit(loginKey, 8, 15*time.Minute, false) {
		writeError(w, http.StatusTooManyRequests, "Too many failed login attempts. Please try again later.")
		return
	}
	u, passwordHash, salt, err := s.findCommunityLoginUser(userID)
	if err != nil || !verifyPassword(req.Password, salt, passwordHash) {
		s.checkRateLimit(loginKey, 8, 15*time.Minute, true)
		writeError(w, http.StatusUnauthorized, "Invalid user ID or password.")
		return
	}
	switch strings.ToLower(strings.TrimSpace(u.Status)) {
	case "pending":
		writeError(w, http.StatusForbidden, "\u4f60\u7684\u6ce8\u518c\u7533\u8bf7\u6b63\u5728\u7b49\u5f85\u7ba1\u7406\u5458\u5ba1\u6838\uff0c\u901a\u8fc7\u540e\u624d\u80fd\u767b\u5f55\u3002")
		return
	case "rejected":
		writeError(w, http.StatusForbidden, "\u4f60\u7684\u6ce8\u518c\u7533\u8bf7\u5df2\u88ab\u9a73\u56de\uff0c\u8bf7\u91cd\u65b0\u6ce8\u518c\u3002")
		return
	case "banned", "deleted":
		writeError(w, http.StatusForbidden, "User account is not active.")
		return
	}
	s.clearRateLimit(loginKey)
	u.LastLoginAt = nowISO()
	_, _ = s.db.Exec(`UPDATE community_users SET last_login_at=? WHERE id=?`, u.LastLoginAt, u.ID)
	token, csrf, expires, err := s.issueCommunitySession(u)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create session.")
		return
	}
	hours := s.cfg.UserSessionHours
	if hours < 1 {
		hours = 1
	}
	http.SetCookie(w, &http.Cookie{Name: "user_session", Value: token, Path: "/", HttpOnly: true, Secure: s.cfg.AdminCookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: hours * 3600})
	resp := publicCommunityUser(u)
	resp["csrf_token"] = csrf
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "expires_at": expires.Format(time.RFC3339Nano), "csrf_token": csrf, "user": resp})
}

func (s *Server) handleCommunityLogout(w http.ResponseWriter, r *http.Request) {
	token, _ := s.communitySessionToken(r)
	if token != "" {
		_, _ = s.db.Exec(`DELETE FROM community_sessions WHERE session_token=?`, token)
	}
	http.SetCookie(w, &http.Cookie{Name: "user_session", Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleCommunityMe(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	resp := publicCommunityUser(user)
	resp["csrf_token"] = user.CSRFToken
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "user": resp, "csrf_token": user.CSRFToken})
}

func (s *Server) communityProfileStats(userPK string) map[string]any {
	var articles, followers, following int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM blog_articles WHERE author_id=? AND status='published' AND visibility='public'`, userPK).Scan(&articles)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM social_follows WHERE following_id=?`, userPK).Scan(&followers)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM social_follows WHERE follower_id=?`, userPK).Scan(&following)
	return map[string]any{"articles": articles, "followers": followers, "following": following}
}

func (s *Server) handleGetCommunityUser(w http.ResponseWriter, r *http.Request, raw string) {
	if !s.communityFrontendAllowsRequest(r) {
		s.handleCommunityLoginRequired(w, r)
		return
	}
	pk, err := s.resolveCommunityUserPK(raw)
	if err != nil {
		writeError(w, http.StatusNotFound, "User not found.")
		return
	}
	u, err := s.loadCommunityUserByPK(pk)
	if err != nil || u.Status == "deleted" {
		writeError(w, http.StatusNotFound, "User not found.")
		return
	}
	current, _, loggedIn := s.communityUserFromRequest(r)
	isOwner := loggedIn && current.ID == u.ID
	following := false
	if loggedIn && current.ID != u.ID {
		var count int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM social_follows WHERE follower_id=? AND following_id=?`, current.ID, u.ID).Scan(&count)
		following = count > 0
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": publicCommunityUser(u), "stats": s.communityProfileStats(u.ID), "is_owner": isOwner, "following": following})
}

func (s *Server) handleUpdateCommunityUser(w http.ResponseWriter, r *http.Request, raw string) {
	current, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	pk, err := s.resolveCommunityUserPK(raw)
	if err != nil {
		writeError(w, http.StatusNotFound, "User not found.")
		return
	}
	if current.ID != pk && !s.canModerateCommunity(r, current) {
		writeError(w, http.StatusForbidden, "Only the owner can edit this profile.")
		return
	}
	var req CommunityProfileUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	nickname, err := validateCommunityNickname(req.Nickname)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	bio := strings.TrimSpace(req.Bio)
	if len([]rune(bio)) > 300 {
		writeError(w, http.StatusBadRequest, "Bio must be 300 characters or fewer.")
		return
	}
	_, err = s.db.Exec(`UPDATE community_users SET nickname=?, avatar_url=?, bio=?, updated_at=? WHERE id=?`, nickname, strings.TrimSpace(req.AvatarURL), bio, nowISO(), pk)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save user profile.")
		return
	}
	u, _ := s.loadCommunityUserByPK(pk)
	writeJSON(w, http.StatusOK, map[string]any{"user": publicCommunityUser(u), "stats": s.communityProfileStats(pk), "is_owner": true})
}

func (s *Server) handleUpdateCommunityAccount(w http.ResponseWriter, r *http.Request) {
	current, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	var req CommunityAccountUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	nickname, err := validateCommunityNickname(req.Nickname)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	userID, err := validateCommunityUserID(firstNonEmpty(req.UserID, current.UserID))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
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
	if userID != current.UserID {
		var exists int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM community_users WHERE user_id=? AND id!=?`, userID, current.ID).Scan(&exists)
		if exists > 0 {
			writeError(w, http.StatusConflict, "User ID already exists.")
			return
		}
	}
	_, err = s.db.Exec(`UPDATE community_users SET user_id=?, nickname=?, avatar_url=?, bio=?, updated_at=? WHERE id=?`, userID, nickname, avatarURL, bio, nowISO(), current.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save account settings.")
		return
	}
	u, _ := s.loadCommunityUserByPK(current.ID)
	resp := publicCommunityUser(u)
	resp["csrf_token"] = current.CSRFToken
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "user": resp, "stats": s.communityProfileStats(current.ID)})
}

func (s *Server) handleUpdateCommunityPassword(w http.ResponseWriter, r *http.Request) {
	current, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	var req CommunityPasswordUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "Old password and new password are required.")
		return
	}
	var passwordHash, salt string
	if err := s.db.QueryRow(`SELECT password_hash, salt FROM community_users WHERE id=?`, current.ID).Scan(&passwordHash, &salt); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load account.")
		return
	}
	if !verifyPassword(req.OldPassword, salt, passwordHash) {
		writeError(w, http.StatusUnauthorized, "Old password is incorrect.")
		return
	}
	if err := validateCommunityPassword(req.NewPassword); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	newSalt, newHash := hashPassword(req.NewPassword, "")
	if _, err := s.db.Exec(`UPDATE community_users SET salt=?, password_hash=?, updated_at=? WHERE id=?`, newSalt, newHash, nowISO(), current.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update password.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
