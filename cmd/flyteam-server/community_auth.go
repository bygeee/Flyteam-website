package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type CommunityUser struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	Nickname     string `json:"nickname"`
	PasswordHash string `json:"password_hash"`
	Salt         string `json:"salt"`
	AvatarURL    string `json:"avatar_url"`
	Bio          string `json:"bio"`
	Role         string `json:"role"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	LastLoginAt  string `json:"last_login_at"`
}

type CommunityUserStore struct {
	Users []CommunityUser `json:"users"`
}

type CommunitySession struct {
	ID        string
	UserID    string
	Nickname  string
	Role      string
	Status    string
	CSRFToken string
	ExpiresAt time.Time
}

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

type PublicCommunityUser struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
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

var communityUserIDRe = regexp.MustCompile(`^[0-9A-Za-z_\-]{3,32}$`)

func publicCommunityUser(u CommunityUser) PublicCommunityUser {
	return PublicCommunityUser{ID: u.ID, UserID: u.UserID, Nickname: u.Nickname, AvatarURL: u.AvatarURL, Bio: u.Bio, Role: defaultString(u.Role, "user"), Status: defaultString(u.Status, "active"), CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt, LastLoginAt: u.LastLoginAt}
}

func publicCommunitySession(sess CommunitySession) PublicCommunityUser {
	return PublicCommunityUser{ID: sess.ID, UserID: sess.UserID, Nickname: sess.Nickname, Role: defaultString(sess.Role, "user"), Status: defaultString(sess.Status, "active"), CSRFToken: sess.CSRFToken}
}

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
	if strings.TrimSpace(password) == password && strings.Contains(strings.ToLower(password), "password") {
		return errors.New("Password is too weak.")
	}
	return nil
}

func (s *Server) loadCommunityUsers() (CommunityUserStore, error) {
	if s.db != nil {
		return s.loadCommunityUsersDB()
	}
	store := s.loadCommunityUsersFromJSON()
	out := CommunityUserStore{Users: []CommunityUser{}}
	seen := map[string]bool{}
	for _, u := range store.Users {
		u.UserID = strings.ToLower(strings.TrimSpace(u.UserID))
		u.Nickname = strings.TrimSpace(u.Nickname)
		u.Salt = strings.TrimSpace(u.Salt)
		u.PasswordHash = strings.TrimSpace(u.PasswordHash)
		if u.UserID == "" || u.Nickname == "" || u.Salt == "" || u.PasswordHash == "" || seen[u.UserID] {
			continue
		}
		seen[u.UserID] = true
		if u.ID == "" {
			u.ID = randomHex(6)
		}
		u.Role = defaultString(strings.TrimSpace(u.Role), "user")
		u.Status = defaultString(strings.TrimSpace(u.Status), "active")
		if u.CreatedAt == "" {
			u.CreatedAt = nowISO()
		}
		out.Users = append(out.Users, u)
	}
	return out, nil
}

func (s *Server) loadCommunityUsersFromJSON() CommunityUserStore {
	var store CommunityUserStore
	b, err := os.ReadFile(s.cfg.CommunityUsersFile)
	if err == nil && len(b) > 0 {
		_ = json.Unmarshal(b, &store)
	}
	return store
}

func (s *Server) loadCommunityUsersDB() (CommunityUserStore, error) {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM community_users`).Scan(&count)
	if count == 0 {
		legacy := s.loadCommunityUsersFromJSON()
		if len(legacy.Users) > 0 {
			_ = s.saveCommunityUsersDB(legacy)
		}
	}
	rows, err := s.db.Query(`SELECT id, user_id, nickname, password_hash, salt, COALESCE(avatar_url,''), COALESCE(bio,''), COALESCE(role,'user'), COALESCE(status,'active'), created_at, COALESCE(updated_at,''), COALESCE(last_login_at,'') FROM community_users ORDER BY created_at ASC, user_id ASC`)
	if err != nil {
		return CommunityUserStore{Users: []CommunityUser{}}, err
	}
	defer rows.Close()
	out := CommunityUserStore{Users: []CommunityUser{}}
	for rows.Next() {
		var u CommunityUser
		if err := rows.Scan(&u.ID, &u.UserID, &u.Nickname, &u.PasswordHash, &u.Salt, &u.AvatarURL, &u.Bio, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt); err != nil {
			return out, err
		}
		u.UserID = strings.ToLower(strings.TrimSpace(u.UserID))
		u.Nickname = strings.TrimSpace(u.Nickname)
		u.Role = defaultString(strings.TrimSpace(u.Role), "user")
		u.Status = defaultString(strings.TrimSpace(u.Status), "active")
		if u.UserID == "" || u.Nickname == "" || strings.TrimSpace(u.Salt) == "" || strings.TrimSpace(u.PasswordHash) == "" {
			continue
		}
		out.Users = append(out.Users, u)
	}
	return out, rows.Err()
}

func (s *Server) saveCommunityUsers(store CommunityUserStore) error {
	if s.db != nil {
		return s.saveCommunityUsersDB(store)
	}
	for i := range store.Users {
		store.Users[i].UserID = strings.ToLower(strings.TrimSpace(store.Users[i].UserID))
		store.Users[i].Nickname = strings.TrimSpace(store.Users[i].Nickname)
		store.Users[i].Role = defaultString(strings.TrimSpace(store.Users[i].Role), "user")
		store.Users[i].Status = defaultString(strings.TrimSpace(store.Users[i].Status), "active")
	}
	return writeJSONAtomic(s.cfg.CommunityUsersFile, store)
}

func (s *Server) saveCommunityUsersDB(store CommunityUserStore) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT INTO community_users(id, user_id, nickname, password_hash, salt, avatar_url, bio, role, status, created_at, updated_at, last_login_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(user_id) DO UPDATE SET
			nickname=excluded.nickname,
			password_hash=excluded.password_hash,
			salt=excluded.salt,
			avatar_url=excluded.avatar_url,
			bio=excluded.bio,
			role=excluded.role,
			status=excluded.status,
			created_at=excluded.created_at,
			updated_at=excluded.updated_at,
			last_login_at=excluded.last_login_at`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, u := range store.Users {
		u.UserID = strings.ToLower(strings.TrimSpace(u.UserID))
		u.Nickname = strings.TrimSpace(u.Nickname)
		u.Role = defaultString(strings.TrimSpace(u.Role), "user")
		u.Status = defaultString(strings.TrimSpace(u.Status), "active")
		u.Salt = strings.TrimSpace(u.Salt)
		u.PasswordHash = strings.TrimSpace(u.PasswordHash)
		if u.UserID == "" || u.Nickname == "" || u.Salt == "" || u.PasswordHash == "" {
			continue
		}
		if u.ID == "" {
			u.ID = randomHex(6)
		}
		if u.CreatedAt == "" {
			u.CreatedAt = nowISO()
		}
		updatedAt := sql.NullString{String: strings.TrimSpace(u.UpdatedAt), Valid: strings.TrimSpace(u.UpdatedAt) != ""}
		lastLogin := sql.NullString{String: strings.TrimSpace(u.LastLoginAt), Valid: strings.TrimSpace(u.LastLoginAt) != ""}
		if _, err := stmt.Exec(u.ID, u.UserID, u.Nickname, u.PasswordHash, u.Salt, strings.TrimSpace(u.AvatarURL), strings.TrimSpace(u.Bio), u.Role, u.Status, u.CreatedAt, updatedAt, lastLogin); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func findCommunityUser(store CommunityUserStore, userID string) *CommunityUser {
	needle := strings.ToLower(strings.TrimSpace(userID))
	for i := range store.Users {
		if strings.ToLower(strings.TrimSpace(store.Users[i].UserID)) == needle {
			return &store.Users[i]
		}
	}
	return nil
}

func findCommunityUserByID(store CommunityUserStore, id string) *CommunityUser {
	needle := strings.TrimSpace(id)
	for i := range store.Users {
		if strings.TrimSpace(store.Users[i].ID) == needle {
			return &store.Users[i]
		}
	}
	return nil
}

func publicCommunityProfile(u CommunityUser, articleCount int, isOwner bool) M {
	return M{"user": publicCommunityUser(u), "stats": M{"articles": articleCount, "followers": 0, "following": 0}, "is_owner": isOwner}
}

func (s *Server) countPublishedArticlesByAuthor(authorID string) int {
	store := s.loadBlogArticles()
	count := 0
	for _, article := range store.Articles {
		if article.AuthorID == authorID && article.Status == "published" {
			count++
		}
	}
	return count
}

func (s *Server) issueCommunitySession(u CommunityUser) (string, CommunitySession) {
	token := randomHex(32)
	csrf := randomHex(32)
	hours := s.cfg.UserSessionHours
	if hours < 1 {
		hours = 1
	}
	sess := CommunitySession{ID: u.ID, UserID: u.UserID, Nickname: u.Nickname, Role: defaultString(u.Role, "user"), Status: defaultString(u.Status, "active"), CSRFToken: csrf, ExpiresAt: time.Now().UTC().Add(time.Duration(hours) * time.Hour)}
	s.userSessMu.Lock()
	s.userSessions[token] = sess
	s.userSessMu.Unlock()
	if s.db != nil {
		_ = s.saveCommunitySessionDB(token, sess)
	}
	return token, sess
}

func (s *Server) communityUserFromRequest(r *http.Request) (CommunitySession, bool) {
	if c, err := r.Cookie("user_session"); err == nil {
		return s.communityUserFromToken(c.Value)
	}
	return CommunitySession{}, false
}

func (s *Server) communityUserFromToken(token string) (CommunitySession, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return CommunitySession{}, false
	}
	if s.db != nil {
		return s.loadCommunitySessionDB(token)
	}
	s.userSessMu.Lock()
	defer s.userSessMu.Unlock()
	sess, ok := s.userSessions[token]
	if !ok {
		return CommunitySession{}, false
	}
	if time.Now().UTC().After(sess.ExpiresAt) {
		delete(s.userSessions, token)
		return CommunitySession{}, false
	}
	return sess, true
}

func (s *Server) loadCommunitySessionDB(token string) (CommunitySession, bool) {
	var sess CommunitySession
	var expiresAt string
	err := s.db.QueryRow(`SELECT u.id, u.user_id, u.nickname, COALESCE(u.role,'user'), COALESCE(u.status,'active'), s.csrf_token, s.expires_at
		FROM community_sessions s
		JOIN community_users u ON u.id = s.user_pk
		WHERE s.session_token = ?`, token).Scan(&sess.ID, &sess.UserID, &sess.Nickname, &sess.Role, &sess.Status, &sess.CSRFToken, &expiresAt)
	if err != nil {
		return CommunitySession{}, false
	}
	exp, err := time.Parse(time.RFC3339Nano, expiresAt)
	if err != nil || time.Now().UTC().After(exp) {
		_ = s.deleteCommunitySessionDB(token)
		return CommunitySession{}, false
	}
	sess.ExpiresAt = exp
	return sess, true
}

func (s *Server) saveCommunitySessionDB(token string, sess CommunitySession) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.Exec(`INSERT INTO community_sessions(session_token, user_pk, csrf_token, expires_at, created_at, user_agent_hash, ip_hash)
		VALUES(?,?,?,?,?,?,?)
		ON CONFLICT(session_token) DO UPDATE SET
			user_pk=excluded.user_pk,
			csrf_token=excluded.csrf_token,
			expires_at=excluded.expires_at`, token, sess.ID, sess.CSRFToken, sess.ExpiresAt.UTC().Format(time.RFC3339Nano), nowISO(), nil, nil)
	return err
}

func (s *Server) deleteCommunitySessionDB(token string) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.Exec(`DELETE FROM community_sessions WHERE session_token = ?`, token)
	return err
}

func (s *Server) requireCommunityUser(w http.ResponseWriter, r *http.Request) (CommunitySession, bool) {
	user, ok := s.communityUserFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "User login required.")
		return CommunitySession{}, false
	}
	if user.Status != "active" {
		writeError(w, http.StatusForbidden, "User account is not active.")
		return CommunitySession{}, false
	}
	return user, true
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
	store, _ := s.loadCommunityUsers()
	if findCommunityUser(store, userID) != nil {
		writeError(w, http.StatusConflict, "User ID already exists.")
		return
	}
	salt, hash := hashPassword(req.Password, "")
	u := CommunityUser{ID: randomHex(6), UserID: userID, Nickname: nickname, PasswordHash: hash, Salt: salt, Role: "user", Status: "active", CreatedAt: nowISO()}
	store.Users = append(store.Users, u)
	if err := s.saveCommunityUsers(store); err != nil {
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
	store, _ := s.loadCommunityUsers()
	u := findCommunityUser(store, userID)
	if u == nil || !verifyPassword(req.Password, u.Salt, u.PasswordHash) {
		s.checkRateLimit(loginKey, 8, 15*time.Minute, true)
		writeError(w, http.StatusUnauthorized, "Invalid user ID or password.")
		return
	}
	if defaultString(u.Status, "active") != "active" {
		writeError(w, http.StatusForbidden, "User account is not active.")
		return
	}
	s.clearRateLimit(loginKey)
	for i := range store.Users {
		if store.Users[i].ID == u.ID {
			store.Users[i].LastLoginAt = nowISO()
			*u = store.Users[i]
		}
	}
	_ = s.saveCommunityUsers(store)
	token, sess := s.issueCommunitySession(*u)
	http.SetCookie(w, &http.Cookie{Name: "user_session", Value: token, Path: "/", HttpOnly: true, Secure: s.cfg.AdminCookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: max(1, s.cfg.UserSessionHours) * 3600})
	writeJSON(w, http.StatusOK, map[string]any{"expires_at": sess.ExpiresAt.Format(time.RFC3339Nano), "csrf_token": sess.CSRFToken, "user": publicCommunityUser(*u)})
}

func (s *Server) handleCommunityLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("user_session"); err == nil {
		s.userSessMu.Lock()
		delete(s.userSessions, c.Value)
		s.userSessMu.Unlock()
		_ = s.deleteCommunitySessionDB(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "user_session", Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleCommunityMe(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "user": publicCommunitySession(user), "csrf_token": user.CSRFToken})
}

func (s *Server) handleGetCommunityUser(w http.ResponseWriter, r *http.Request, id string) {
	store, _ := s.loadCommunityUsers()
	u := findCommunityUser(store, id)
	if u == nil {
		u = findCommunityUserByID(store, id)
	}
	if u == nil || defaultString(u.Status, "active") == "deleted" {
		writeError(w, http.StatusNotFound, "User not found.")
		return
	}
	current, loggedIn := s.communityUserFromRequest(r)
	isOwner := loggedIn && current.ID == u.ID
	writeJSON(w, http.StatusOK, publicCommunityProfile(*u, s.countPublishedArticlesByAuthor(u.ID), isOwner))
}

func (s *Server) handleUpdateCommunityUser(w http.ResponseWriter, r *http.Request, id string) {
	current, ok := s.requireCommunityUser(w, r)
	if !ok {
		return
	}
	store, _ := s.loadCommunityUsers()
	u := findCommunityUser(store, id)
	if u == nil {
		u = findCommunityUserByID(store, id)
	}
	if u == nil {
		writeError(w, http.StatusNotFound, "User not found.")
		return
	}
	if current.ID != u.ID {
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
	u.Nickname = nickname
	u.AvatarURL = strings.TrimSpace(req.AvatarURL)
	u.Bio = bio
	u.UpdatedAt = nowISO()
	if err := s.saveCommunityUsers(store); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save user profile.")
		return
	}
	writeJSON(w, http.StatusOK, publicCommunityProfile(*u, s.countPublishedArticlesByAuthor(u.ID), true))
}
