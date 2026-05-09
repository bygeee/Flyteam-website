package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const passwordIterations = 260000

type AdminUser struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	Role         string `json:"role"`
	Salt         string `json:"salt"`
	PasswordHash string `json:"password_hash"`
	CreatedAt    string `json:"created_at"`
	LastLoginAt  string `json:"last_login_at"`
}

type AdminStore struct {
	Users []AdminUser `json:"users"`
}

type AdminSession struct {
	ID          string
	Username    string
	DisplayName string
	Role        string
	CSRFToken   string
	ExpiresAt   time.Time
}

type AdminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type AdminUserCreateRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}
type AdminPasswordUpdateRequest struct {
	Password string `json:"password"`
}
type AdminRoleUpdateRequest struct {
	Role string `json:"role"`
}

type PublicAdmin struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	CreatedAt   string `json:"created_at"`
	LastLoginAt string `json:"last_login_at"`
	CSRFToken   string `json:"csrf_token,omitempty"`
}

func normalizeRole(role string) string {
	if strings.ToLower(strings.TrimSpace(role)) == "superadmin" {
		return "superadmin"
	}
	return "admin"
}

func publicAdmin(u AdminUser) PublicAdmin {
	return PublicAdmin{ID: u.ID, Username: u.Username, DisplayName: u.DisplayName, Role: normalizeRole(u.Role), CreatedAt: u.CreatedAt, LastLoginAt: u.LastLoginAt}
}

func publicSession(s AdminSession) PublicAdmin {
	return PublicAdmin{ID: s.ID, Username: s.Username, DisplayName: s.DisplayName, Role: normalizeRole(s.Role), CSRFToken: s.CSRFToken}
}

func (s *Server) loadAdminUsers() (AdminStore, error) {
	if s.db != nil {
		return s.loadAdminUsersDB()
	}
	store := s.loadAdminUsersFromJSON()
	out := AdminStore{Users: []AdminUser{}}
	for _, u := range store.Users {
		u.Username = strings.TrimSpace(u.Username)
		u.Salt = strings.TrimSpace(u.Salt)
		u.PasswordHash = strings.TrimSpace(u.PasswordHash)
		if u.Username == "" || u.Salt == "" || u.PasswordHash == "" {
			continue
		}
		if u.ID == "" {
			u.ID = randomHex(6)
		}
		u.DisplayName = strings.TrimSpace(u.DisplayName)
		u.Role = normalizeRole(u.Role)
		if u.CreatedAt == "" {
			u.CreatedAt = nowISO()
		}
		out.Users = append(out.Users, u)
	}
	if len(out.Users) == 0 {
		pass := s.cfg.AdminPassword
		if pass == "" {
			pass = s.cfg.AdminToken
		}
		if pass == "" {
			pass = "admin123456"
		}
		salt, hash := hashPassword(pass, "")
		out.Users = []AdminUser{{ID: randomHex(6), Username: "admin", DisplayName: "System Admin", Role: "admin", Salt: salt, PasswordHash: hash, CreatedAt: nowISO()}}
		_ = s.saveAdminUsers(out)
	}
	return out, nil
}

func (s *Server) saveAdminUsers(store AdminStore) error {
	if s.db != nil {
		return s.saveAdminUsersDB(store)
	}
	for i := range store.Users {
		store.Users[i].Role = normalizeRole(store.Users[i].Role)
	}
	return writeJSONAtomic(s.cfg.AdminUsersFile, store)
}

func (s *Server) loadAdminUsersFromJSON() AdminStore {
	var store AdminStore
	b, err := os.ReadFile(s.cfg.AdminUsersFile)
	if err == nil && len(b) > 0 {
		_ = json.Unmarshal(b, &store)
	}
	return store
}

func (s *Server) loadAdminUsersDB() (AdminStore, error) {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM admin_users`).Scan(&count)
	if count == 0 {
		legacy := s.loadAdminUsersFromJSON()
		if len(legacy.Users) > 0 {
			_ = s.saveAdminUsersDB(legacy)
		}
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM admin_users`).Scan(&count)
	}
	if count == 0 {
		pass := s.cfg.AdminPassword
		if pass == "" {
			pass = s.cfg.AdminToken
		}
		if pass == "" {
			pass = "admin123456"
		}
		salt, hash := hashPassword(pass, "")
		u := AdminUser{ID: randomHex(6), Username: "admin", DisplayName: "System Admin", Role: "admin", Salt: salt, PasswordHash: hash, CreatedAt: nowISO()}
		if err := s.saveAdminUsersDB(AdminStore{Users: []AdminUser{u}}); err != nil {
			return AdminStore{}, err
		}
	}
	rows, err := s.db.Query(`SELECT id, username, COALESCE(display_name,''), role, salt, password_hash, created_at, COALESCE(last_login_at,'') FROM admin_users ORDER BY created_at ASC, username ASC`)
	if err != nil {
		return AdminStore{}, err
	}
	defer rows.Close()
	out := AdminStore{Users: []AdminUser{}}
	for rows.Next() {
		var u AdminUser
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.Salt, &u.PasswordHash, &u.CreatedAt, &u.LastLoginAt); err != nil {
			return out, err
		}
		u.Username = strings.TrimSpace(u.Username)
		u.Role = normalizeRole(u.Role)
		if u.ID == "" || u.Username == "" || u.Salt == "" || u.PasswordHash == "" {
			continue
		}
		out.Users = append(out.Users, u)
	}
	return out, rows.Err()
}

func (s *Server) saveAdminUsersDB(store AdminStore) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM admin_users`); err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO admin_users(id, username, display_name, role, salt, password_hash, created_at, last_login_at) VALUES(?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, u := range store.Users {
		u.Username = strings.TrimSpace(u.Username)
		u.Salt = strings.TrimSpace(u.Salt)
		u.PasswordHash = strings.TrimSpace(u.PasswordHash)
		if u.Username == "" || u.Salt == "" || u.PasswordHash == "" {
			continue
		}
		if u.ID == "" {
			u.ID = randomHex(6)
		}
		if u.CreatedAt == "" {
			u.CreatedAt = nowISO()
		}
		lastLogin := sql.NullString{String: strings.TrimSpace(u.LastLoginAt), Valid: strings.TrimSpace(u.LastLoginAt) != ""}
		if _, err := stmt.Exec(u.ID, u.Username, strings.TrimSpace(u.DisplayName), normalizeRole(u.Role), u.Salt, u.PasswordHash, u.CreatedAt, lastLogin); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func findAdmin(store AdminStore, username string) *AdminUser {
	needle := strings.ToLower(strings.TrimSpace(username))
	for i := range store.Users {
		if strings.ToLower(strings.TrimSpace(store.Users[i].Username)) == needle {
			return &store.Users[i]
		}
	}
	return nil
}

func hashPassword(password, saltHex string) (string, string) {
	if saltHex == "" {
		saltHex = randomHex(16)
	}
	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		salt = []byte(saltHex)
	}
	dk := pbkdf2SHA256([]byte(password), salt, passwordIterations, 32)
	return saltHex, hex.EncodeToString(dk)
}

func verifyPassword(password, salt, expected string) bool {
	_, got := hashPassword(password, salt)
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}

func pbkdf2SHA256(password, salt []byte, iter, keyLen int) []byte {
	hLen := 32
	nBlocks := (keyLen + hLen - 1) / hLen
	var out []byte
	for block := 1; block <= nBlocks; block++ {
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		t := make([]byte, len(u))
		copy(t, u)
		for i := 1; i < iter; i++ {
			mac = hmac.New(sha256.New, password)
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}

func (s *Server) adminFromRequest(r *http.Request) (AdminSession, bool) {
	if tok := strings.TrimSpace(r.Header.Get("X-Admin-Token")); tok != "" {
		return s.adminFromToken(tok)
	}
	if c, err := r.Cookie("admin_session"); err == nil {
		return s.adminFromToken(c.Value)
	}
	return AdminSession{}, false
}

func (s *Server) adminFromToken(token string) (AdminSession, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return AdminSession{}, false
	}
	if s.cfg.AdminToken != "" && hmac.Equal([]byte(token), []byte(s.cfg.AdminToken)) {
		return AdminSession{ID: "legacy-token", Username: "legacy-admin", Role: "superadmin", DisplayName: "Token Super Admin"}, true
	}
	if s.db != nil {
		var payload struct {
			ID          string `json:"id"`
			Username    string `json:"username"`
			DisplayName string `json:"display_name"`
			Role        string `json:"role"`
			CSRFToken   string `json:"csrf_token"`
			ExpiresAt   string `json:"expires_at"`
		}
		if s.loadCacheJSON("admin_session", token, &payload) {
			expiresAt, ok := parseCacheTime(payload.ExpiresAt)
			if ok && time.Now().UTC().Before(expiresAt) {
				return AdminSession{ID: payload.ID, Username: payload.Username, DisplayName: payload.DisplayName, Role: normalizeRole(payload.Role), CSRFToken: payload.CSRFToken, ExpiresAt: expiresAt}, true
			}
			s.deleteCache("admin_session", token)
		}
	}
	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	sess, ok := s.sessions[token]
	if !ok {
		return AdminSession{}, false
	}
	if time.Now().UTC().After(sess.ExpiresAt) {
		delete(s.sessions, token)
		return AdminSession{}, false
	}
	return sess, true
}

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) (AdminSession, bool) {
	admin, ok := s.adminFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized admin action.")
		return AdminSession{}, false
	}
	return admin, true
}

func (s *Server) requireSuperAdmin(w http.ResponseWriter, r *http.Request) (AdminSession, bool) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return AdminSession{}, false
	}
	if normalizeRole(admin.Role) != "superadmin" {
		writeError(w, http.StatusForbidden, "Super administrator permission required.")
		return AdminSession{}, false
	}
	return admin, true
}

func (s *Server) issueAdminSession(u AdminUser) (string, AdminSession) {
	token := randomHex(32)
	csrf := randomHex(32)
	hours := s.cfg.AdminSessionHours
	if hours < 1 {
		hours = 1
	}
	sess := AdminSession{ID: u.ID, Username: u.Username, DisplayName: u.DisplayName, Role: normalizeRole(u.Role), CSRFToken: csrf, ExpiresAt: time.Now().UTC().Add(time.Duration(hours) * time.Hour)}
	if s.db != nil {
		_ = s.saveCacheJSON("admin_session", token, map[string]any{"id": sess.ID, "username": sess.Username, "display_name": sess.DisplayName, "role": sess.Role, "csrf_token": sess.CSRFToken, "expires_at": sess.ExpiresAt.Format(time.RFC3339Nano)}, sess.ExpiresAt)
	} else {
		s.sessMu.Lock()
		s.sessions[token] = sess
		s.sessMu.Unlock()
	}
	return token, sess
}

func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	var req AdminLoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return
	}
	username := strings.TrimSpace(req.Username)
	if username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "Username and password are required.")
		return
	}
	loginKey := "login:" + clientIP(r) + ":" + strings.ToLower(username)
	if !s.checkRateLimit(loginKey, 8, 15*time.Minute, false) {
		writeError(w, http.StatusTooManyRequests, "Too many failed login attempts. Please try again later.")
		return
	}
	store, _ := s.loadAdminUsers()
	u := findAdmin(store, username)
	if u == nil || !verifyPassword(req.Password, u.Salt, u.PasswordHash) {
		s.checkRateLimit(loginKey, 8, 15*time.Minute, true)
		writeError(w, http.StatusUnauthorized, "Invalid username or password.")
		return
	}
	s.clearRateLimit(loginKey)
	for i := range store.Users {
		if store.Users[i].ID == u.ID {
			store.Users[i].LastLoginAt = nowISO()
			*u = store.Users[i]
		}
	}
	_ = s.saveAdminUsers(store)
	token, sess := s.issueAdminSession(*u)
	http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: token, Path: "/", HttpOnly: true, Secure: s.cfg.AdminCookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: max(1, s.cfg.AdminSessionHours) * 3600})
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "expires_at": sess.ExpiresAt.Format(time.RFC3339Nano), "csrf_token": sess.CSRFToken, "user": publicAdmin(*u)})
}

func (s *Server) handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	for _, token := range []string{r.Header.Get("X-Admin-Token")} {
		if token != "" {
			s.deleteCache("admin_session", token)
			s.sessMu.Lock()
			delete(s.sessions, token)
			s.sessMu.Unlock()
		}
	}
	if c, err := r.Cookie("admin_session"); err == nil {
		s.deleteCache("admin_session", c.Value)
		s.sessMu.Lock()
		delete(s.sessions, c.Value)
		s.sessMu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminPing(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "user": publicSession(admin), "csrf_token": admin.CSRFToken})
}

func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	store, _ := s.loadAdminUsers()
	users := []PublicAdmin{}
	for _, u := range store.Users {
		users = append(users, publicAdmin(u))
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": users})
}

var usernameRe = regexp.MustCompile(`^[0-9A-Za-z_@.\-]{3,40}$`)

func validateUsername(username string) (string, error) {
	clean := strings.TrimSpace(username)
	if !usernameRe.MatchString(clean) {
		return "", errors.New("Username must be 3-40 chars: letters, numbers, _, ., -, @.")
	}
	return clean, nil
}

func (s *Server) handleAddAdminUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	var req AdminUserCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, 400, "Invalid JSON.")
		return
	}
	username, err := validateUsername(req.Username)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if len(req.Password) < 6 {
		writeError(w, 400, "Password must be at least 6 characters.")
		return
	}
	store, _ := s.loadAdminUsers()
	if findAdmin(store, username) != nil {
		writeError(w, 409, "Admin username already exists.")
		return
	}
	salt, hash := hashPassword(req.Password, "")
	u := AdminUser{ID: randomHex(6), Username: username, DisplayName: strings.TrimSpace(req.DisplayName), Role: normalizeRole(req.Role), Salt: salt, PasswordHash: hash, CreatedAt: nowISO()}
	store.Users = append(store.Users, u)
	_ = s.saveAdminUsers(store)
	writeJSON(w, 201, map[string]any{"user": publicAdmin(u)})
}

func (s *Server) handleUpdateAdminPassword(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	var req AdminPasswordUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, 400, "Invalid JSON.")
		return
	}
	if len(req.Password) < 6 {
		writeError(w, 400, "Password must be at least 6 characters.")
		return
	}
	store, _ := s.loadAdminUsers()
	found := false
	username := ""
	for i := range store.Users {
		if store.Users[i].ID == id {
			salt, hash := hashPassword(req.Password, "")
			store.Users[i].Salt = salt
			store.Users[i].PasswordHash = hash
			username = store.Users[i].Username
			found = true
		}
	}
	if !found {
		writeError(w, 404, "Admin user not found.")
		return
	}
	_ = s.saveAdminUsers(store)
	s.dropSessionsFor(username)
	writeJSON(w, 200, map[string]any{"updated": id})
}

func (s *Server) handleUpdateAdminRole(w http.ResponseWriter, r *http.Request, id string) {
	admin, ok := s.requireSuperAdmin(w, r)
	if !ok {
		return
	}
	var req AdminRoleUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, 400, "Invalid JSON.")
		return
	}
	next := normalizeRole(req.Role)
	store, _ := s.loadAdminUsers()
	idx := -1
	for i := range store.Users {
		if store.Users[i].ID == id {
			idx = i
		}
	}
	if idx < 0 {
		writeError(w, 404, "Admin user not found.")
		return
	}
	if store.Users[idx].Role == "superadmin" && next != "superadmin" {
		cnt := 0
		for _, u := range store.Users {
			if normalizeRole(u.Role) == "superadmin" {
				cnt++
			}
		}
		if cnt <= 1 {
			writeError(w, 400, "At least one super administrator must remain.")
			return
		}
		if admin.ID == id {
			writeError(w, 400, "You cannot downgrade your current super administrator account.")
			return
		}
	}
	store.Users[idx].Role = next
	target := store.Users[idx]
	_ = s.saveAdminUsers(store)
	s.dropSessionsFor(target.Username)
	writeJSON(w, 200, map[string]any{"user": publicAdmin(target)})
}

func (s *Server) handleDeleteAdminUser(w http.ResponseWriter, r *http.Request, id string) {
	admin, ok := s.requireSuperAdmin(w, r)
	if !ok {
		return
	}
	store, _ := s.loadAdminUsers()
	idx := -1
	for i := range store.Users {
		if store.Users[i].ID == id {
			idx = i
		}
	}
	if idx < 0 {
		writeError(w, 404, "Admin user not found.")
		return
	}
	if len(store.Users) <= 1 {
		writeError(w, 400, "At least one admin user must remain.")
		return
	}
	target := store.Users[idx]
	if normalizeRole(target.Role) == "superadmin" {
		cnt := 0
		for _, u := range store.Users {
			if normalizeRole(u.Role) == "superadmin" {
				cnt++
			}
		}
		if cnt <= 1 {
			writeError(w, 400, "At least one super administrator must remain.")
			return
		}
	}
	if admin.ID == id {
		writeError(w, 400, "You cannot delete the current admin account.")
		return
	}
	store.Users = append(store.Users[:idx], store.Users[idx+1:]...)
	_ = s.saveAdminUsers(store)
	s.dropSessionsFor(target.Username)
	writeJSON(w, 200, map[string]any{"deleted": id})
}

func (s *Server) dropSessionsFor(username string) {
	s.dropAdminSessionCacheFor(username)
	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	for t, sess := range s.sessions {
		if strings.EqualFold(sess.Username, username) {
			delete(s.sessions, t)
		}
	}
}

func (s *Server) dropAdminSessionCacheFor(username string) {
	if s.db == nil {
		return
	}
	rows, err := s.db.Query(`SELECT key, value_json FROM app_cache WHERE scope='admin_session'`)
	if err != nil {
		return
	}
	defer rows.Close()
	keys := []string{}
	for rows.Next() {
		var key, raw string
		if rows.Scan(&key, &raw) != nil {
			continue
		}
		var payload struct {
			Username string `json:"username"`
		}
		if json.Unmarshal([]byte(raw), &payload) == nil && strings.EqualFold(payload.Username, username) {
			keys = append(keys, key)
		}
	}
	for _, key := range keys {
		s.deleteCache("admin_session", key)
	}
}
