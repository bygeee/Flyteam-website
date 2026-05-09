package main

import (
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

var communityUserIDRe = regexp.MustCompile(`^[0-9A-Za-z_\-]{3,32}$`)

func validateCommunityUserID(userID string) (string, error) {
	clean := strings.ToLower(strings.TrimSpace(userID))
	if !communityUserIDRe.MatchString(clean) {
		return "", errors.New("User ID must be 3-32 chars: letters, numbers, _, -.")
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
	var req CommunityRegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	nickname, err := validateCommunityNickname(req.Nickname)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	userID, err := validateCommunityUserID(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateCommunityPassword(req.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var exists int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM community_users WHERE user_id=?`, userID).Scan(&exists)
	if exists > 0 {
		writeError(w, http.StatusConflict, "User ID already exists.")
		return
	}
	salt, hash := hashPassword(req.Password, "")
	u := CommunityUser{ID: randomHex(6), UserID: userID, Nickname: nickname, Role: "user", Status: "active", CreatedAt: nowISO()}
	_, err = s.db.Exec(`INSERT INTO community_users(id, user_id, nickname, password_hash, salt, role, status, created_at) VALUES(?,?,?,?,?,?,?,?)`, u.ID, u.UserID, u.Nickname, hash, salt, u.Role, u.Status, u.CreatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save user.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": publicCommunityUser(u)})
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
	if u.Status == "banned" || u.Status == "deleted" {
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
