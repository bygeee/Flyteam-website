package app

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"
)

type CommunityUser struct {
	ID          string `json:"user_pk"`
	UserID      string `json:"id"`
	Nickname    string `json:"nickname"`
	AvatarURL   string `json:"avatar_url"`
	Bio         string `json:"bio"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	LastLoginAt string `json:"last_login_at"`
	CSRFToken   string `json:"csrf_token,omitempty"`
}

func publicCommunityUser(u CommunityUser) map[string]any {
	return map[string]any{
		"id":            u.UserID,
		"user_id":       u.UserID,
		"user_pk":       u.ID,
		"nickname":      u.Nickname,
		"avatar_url":    u.AvatarURL,
		"bio":           u.Bio,
		"role":          u.Role,
		"status":        u.Status,
		"created_at":    u.CreatedAt,
		"updated_at":    u.UpdatedAt,
		"last_login_at": u.LastLoginAt,
	}
}

func (s *Server) communitySessionToken(r *http.Request) (token, source string) {
	if tok := strings.TrimSpace(r.Header.Get("X-User-Token")); tok != "" {
		return tok, "header"
	}
	if auth := strings.TrimSpace(r.Header.Get("Authorization")); strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:]), "header"
	}
	if c, err := r.Cookie("user_session"); err == nil && strings.TrimSpace(c.Value) != "" {
		return strings.TrimSpace(c.Value), "cookie"
	}
	return "", ""
}

func (s *Server) communityUserFromRequest(r *http.Request) (CommunityUser, string, bool) {
	if s.db == nil {
		return CommunityUser{}, "", false
	}
	token, source := s.communitySessionToken(r)
	if token == "" {
		return CommunityUser{}, "", false
	}
	var u CommunityUser
	var expiresAt string
	err := s.db.QueryRow(`SELECT u.id, u.user_id, u.nickname, COALESCE(u.avatar_url,''), COALESCE(u.bio,''), u.role, u.status, u.created_at, COALESCE(u.updated_at,''), COALESCE(u.last_login_at,''), s.csrf_token, s.expires_at
		FROM community_sessions s JOIN community_users u ON u.id=s.user_pk
		WHERE s.session_token=?`, token).Scan(&u.ID, &u.UserID, &u.Nickname, &u.AvatarURL, &u.Bio, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt, &u.CSRFToken, &expiresAt)
	if err != nil {
		return CommunityUser{}, source, false
	}
	if sessionExpired(expiresAt) {
		_, _ = s.db.Exec(`DELETE FROM community_sessions WHERE session_token=?`, token)
		return CommunityUser{}, source, false
	}
	return u, source, true
}

func sessionExpired(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return true
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return time.Now().UTC().After(t)
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return time.Now().UTC().After(t)
	}
	return true
}

func (s *Server) requireCommunityUser(w http.ResponseWriter, r *http.Request) (CommunityUser, bool) {
	u, source, ok := s.communityUserFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "User login required.")
		return CommunityUser{}, false
	}
	switch strings.ToLower(strings.TrimSpace(u.Status)) {
	case "pending":
		writeError(w, http.StatusForbidden, "This user account is waiting for administrator approval.")
		return CommunityUser{}, false
	case "rejected":
		writeError(w, http.StatusForbidden, "This registration was rejected. Please register again.")
		return CommunityUser{}, false
	case "banned", "deleted":
		writeError(w, http.StatusForbidden, "This user account is not allowed to use community features.")
		return CommunityUser{}, false
	}
	if source == "cookie" && isMutating(r.Method) && u.CSRFToken != "" && r.Header.Get("X-CSRF-Token") != u.CSRFToken {
		writeError(w, http.StatusForbidden, "CSRF token missing or invalid.")
		return CommunityUser{}, false
	}
	return u, true
}

func (s *Server) requireCommunityWriter(w http.ResponseWriter, r *http.Request) (CommunityUser, bool) {
	u, ok := s.requireCommunityUser(w, r)
	if !ok {
		return CommunityUser{}, false
	}
	if u.Status == "muted" {
		writeError(w, http.StatusForbidden, "This user is muted.")
		return CommunityUser{}, false
	}
	return u, true
}

func (s *Server) canModerateCommunity(r *http.Request, u CommunityUser) bool {
	if admin, ok := s.adminFromRequest(r); ok && canManageBlogRole(admin.Role) {
		return true
	}
	role := strings.ToLower(strings.TrimSpace(u.Role))
	return role == "moderator" || role == "admin" || role == "superadmin"
}

func (s *Server) resolveCommunityUserPK(raw string) (string, error) {
	if s.db == nil {
		return "", errors.New("database unavailable")
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("user id is required")
	}
	var id string
	err := s.db.QueryRow(`SELECT id FROM community_users WHERE (id=? OR user_id=?) AND status NOT IN ('deleted','pending','rejected')`, raw, raw).Scan(&id)
	if err == sql.ErrNoRows {
		return "", errors.New("user not found")
	}
	return id, err
}

func (s *Server) loadCommunityUserByPK(id string) (CommunityUser, error) {
	var u CommunityUser
	err := s.db.QueryRow(`SELECT id, user_id, nickname, COALESCE(avatar_url,''), COALESCE(bio,''), role, status, created_at, COALESCE(updated_at,''), COALESCE(last_login_at,'')
		FROM community_users WHERE id=? AND status NOT IN ('deleted','pending','rejected')`, id).Scan(&u.ID, &u.UserID, &u.Nickname, &u.AvatarURL, &u.Bio, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt)
	return u, err
}

func hashForAudit(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
