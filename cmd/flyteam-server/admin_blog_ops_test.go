package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func adminAPIToken(s *Server, role string) string {
	token := "admin-token-" + role
	s.sessions[token] = AdminSession{ID: role + "-id", Username: role, DisplayName: role, Role: role, ExpiresAt: time.Now().UTC().Add(time.Hour)}
	return token
}

func adminAPIReq(s *Server, method, path, token string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.RemoteAddr = "127.0.0.1:12345"
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("X-Admin-Token", token)
	}
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	return rr
}

func TestAdminCommunityUsersAndSuperAuditPermissions(t *testing.T) {
	s := newDLTestServer(t)
	seedDLUser(t, s.db, "u-a", "alice", "Alice")
	seedDLUser(t, s.db, "u-b", "bob", "Bob")
	now := nowISO()
	if _, err := s.db.Exec(`INSERT INTO private_conversations(id,user_a,user_b,created_at,updated_at,last_message_at) VALUES('conv1','u-a','u-b',?,?,?)`, now, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO private_messages(id,conversation_id,sender_id,content,status,created_at) VALUES('pm1','conv1','u-a','hello','normal',?)`, now); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO chat_groups(id,owner_id,name,visibility,created_at,updated_at) VALUES('grp1','u-a','security group','public',?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO chat_group_members(group_id,user_id,role,status,joined_at) VALUES('grp1','u-a','owner','active',?)`, now); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO chat_group_messages(id,group_id,sender_id,content,status,created_at) VALUES('gm1','grp1','u-a','hello group','normal',?)`, now); err != nil {
		t.Fatal(err)
	}

	blogToken := adminAPIToken(s, "blog_admin")
	list := adminAPIReq(s, http.MethodGet, "/api/admin/community/users", blogToken, nil)
	if list.Code != http.StatusOK {
		t.Fatalf("blog admin list status=%d body=%s", list.Code, list.Body.String())
	}
	mute := adminAPIReq(s, http.MethodPut, "/api/admin/community/users/alice/status", blogToken, map[string]any{"status": "muted"})
	if mute.Code != http.StatusOK {
		t.Fatalf("blog admin mute status=%d body=%s", mute.Code, mute.Body.String())
	}
	grantSuper := adminAPIReq(s, http.MethodPut, "/api/admin/community/users/alice/role", blogToken, map[string]any{"role": "superadmin"})
	if grantSuper.Code != http.StatusForbidden {
		t.Fatalf("blog admin should not grant super role, status=%d body=%s", grantSuper.Code, grantSuper.Body.String())
	}
	auditDenied := adminAPIReq(s, http.MethodGet, "/api/superadmin/audit/private-conversations", blogToken, nil)
	if auditDenied.Code != http.StatusForbidden {
		t.Fatalf("blog admin audit status=%d body=%s", auditDenied.Code, auditDenied.Body.String())
	}

	siteToken := adminAPIToken(s, "site_admin")
	siteCommunityList := adminAPIReq(s, http.MethodGet, "/api/admin/community/users", siteToken, nil)
	if siteCommunityList.Code != http.StatusForbidden {
		t.Fatalf("site admin must not manage blog users, status=%d body=%s", siteCommunityList.Code, siteCommunityList.Body.String())
	}
	siteRecruitList := adminAPIReq(s, http.MethodGet, "/api/recruit/list", siteToken, nil)
	if siteRecruitList.Code != http.StatusOK {
		t.Fatalf("site admin recruit list status=%d body=%s", siteRecruitList.Code, siteRecruitList.Body.String())
	}
	blogRecruitList := adminAPIReq(s, http.MethodGet, "/api/recruit/list", blogToken, nil)
	if blogRecruitList.Code != http.StatusForbidden {
		t.Fatalf("blog admin must not manage recruit list, status=%d body=%s", blogRecruitList.Code, blogRecruitList.Body.String())
	}
	publicStats := adminAPIReq(s, http.MethodGet, "/api/recruit/stats", "", nil)
	if publicStats.Code != http.StatusOK {
		t.Fatalf("recruit stats should remain public, status=%d body=%s", publicStats.Code, publicStats.Body.String())
	}

	superToken := adminAPIToken(s, "superadmin")
	conv := adminAPIReq(s, http.MethodGet, "/api/superadmin/audit/private-conversations", superToken, nil)
	if conv.Code != http.StatusOK {
		t.Fatalf("super conv status=%d body=%s", conv.Code, conv.Body.String())
	}
	msgs := adminAPIReq(s, http.MethodGet, "/api/superadmin/audit/private-conversations/conv1/messages", superToken, nil)
	if msgs.Code != http.StatusOK {
		t.Fatalf("super private messages status=%d body=%s", msgs.Code, msgs.Body.String())
	}
	groups := adminAPIReq(s, http.MethodGet, "/api/superadmin/audit/groups", superToken, nil)
	if groups.Code != http.StatusOK {
		t.Fatalf("super groups status=%d body=%s", groups.Code, groups.Body.String())
	}
	groupMsgs := adminAPIReq(s, http.MethodGet, "/api/superadmin/audit/groups/grp1/messages", superToken, nil)
	if groupMsgs.Code != http.StatusOK {
		t.Fatalf("super group messages status=%d body=%s", groupMsgs.Code, groupMsgs.Body.String())
	}
}

func TestAdminRoleSplitAllowsOnlyOneSuperAdmin(t *testing.T) {
	s := newDLTestServer(t)
	salt, hash := hashPassword("TestPass!2026", "")
	if err := s.saveAdminUsers(AdminStore{Users: []AdminUser{
		{ID: "super1", Username: "z3ghxxx", DisplayName: "Z3", Role: "superadmin", Salt: salt, PasswordHash: hash, CreatedAt: nowISO()},
		{ID: "site1", Username: "site-admin", DisplayName: "Site", Role: "site_admin", Salt: salt, PasswordHash: hash, CreatedAt: nowISO()},
	}}); err != nil {
		t.Fatal(err)
	}
	superToken := adminAPIToken(s, "superadmin")

	secondSuper := adminAPIReq(s, http.MethodPost, "/api/admin/users", superToken, map[string]any{
		"username": "another-super", "password": "pass123456", "role": "superadmin",
	})
	if secondSuper.Code != http.StatusBadRequest {
		t.Fatalf("creating a second superadmin should fail, status=%d body=%s", secondSuper.Code, secondSuper.Body.String())
	}

	promoteSite := adminAPIReq(s, http.MethodPut, "/api/admin/users/site1/role", superToken, map[string]any{"role": "superadmin"})
	if promoteSite.Code != http.StatusBadRequest {
		t.Fatalf("promoting to a second superadmin should fail, status=%d body=%s", promoteSite.Code, promoteSite.Body.String())
	}

	downgradeOnlySuper := adminAPIReq(s, http.MethodPut, "/api/admin/users/super1/role", superToken, map[string]any{"role": "site_admin"})
	if downgradeOnlySuper.Code != http.StatusBadRequest {
		t.Fatalf("downgrading the only superadmin should fail, status=%d body=%s", downgradeOnlySuper.Code, downgradeOnlySuper.Body.String())
	}
}

func registerCommunityForReview(t *testing.T, s *Server, userID string) *httptest.ResponseRecorder {
	t.Helper()
	return dlReq(s, http.MethodPost, "/api/users/register", "", map[string]any{
		"user_id":  userID,
		"nickname": "New " + userID,
		"password": "GoodPass!2026",
	})
}

func TestCommunityRegistrationRequiresAdminApproval(t *testing.T) {
	s := newDLTestServer(t)
	reg := registerCommunityForReview(t, s, "neo")
	if reg.Code != http.StatusAccepted {
		t.Fatalf("register should be accepted for review, status=%d body=%s", reg.Code, reg.Body.String())
	}
	var status string
	if err := s.db.QueryRow(`SELECT status FROM community_users WHERE user_id='neo'`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "pending" {
		t.Fatalf("new registration status=%q, want pending", status)
	}

	loginPending := dlReq(s, http.MethodPost, "/api/users/login", "", map[string]any{"user_id": "neo", "password": "GoodPass!2026"})
	if loginPending.Code != http.StatusForbidden {
		t.Fatalf("pending user must not login, status=%d body=%s", loginPending.Code, loginPending.Body.String())
	}

	siteToken := adminAPIToken(s, "site_admin")
	siteApprove := adminAPIReq(s, http.MethodPut, "/api/admin/community/users/neo/status", siteToken, map[string]any{"status": "active"})
	if siteApprove.Code != http.StatusForbidden {
		t.Fatalf("site admin must not approve blog registrations, status=%d body=%s", siteApprove.Code, siteApprove.Body.String())
	}

	blogToken := adminAPIToken(s, "blog_admin")
	approve := adminAPIReq(s, http.MethodPut, "/api/admin/community/users/neo/status", blogToken, map[string]any{"status": "active"})
	if approve.Code != http.StatusOK {
		t.Fatalf("blog admin approve status=%d body=%s", approve.Code, approve.Body.String())
	}
	if err := s.db.QueryRow(`SELECT status FROM community_users WHERE user_id='neo'`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "active" {
		t.Fatalf("approved registration status=%q, want active", status)
	}
	loginApproved := dlReq(s, http.MethodPost, "/api/users/login", "", map[string]any{"user_id": "neo", "password": "GoodPass!2026"})
	if loginApproved.Code != http.StatusOK {
		t.Fatalf("approved user login status=%d body=%s", loginApproved.Code, loginApproved.Body.String())
	}
}

func TestCommunityRegistrationRejectReleasesUserID(t *testing.T) {
	s := newDLTestServer(t)
	reg := registerCommunityForReview(t, s, "rejectme")
	if reg.Code != http.StatusAccepted {
		t.Fatalf("register should be accepted for review, status=%d body=%s", reg.Code, reg.Body.String())
	}
	blogToken := adminAPIToken(s, "blog_admin")
	reject := adminAPIReq(s, http.MethodPut, "/api/admin/community/users/rejectme/status", blogToken, map[string]any{"status": "rejected"})
	if reject.Code != http.StatusOK {
		t.Fatalf("reject pending registration status=%d body=%s", reject.Code, reject.Body.String())
	}
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM community_users WHERE user_id='rejectme'`).Scan(&count)
	if count != 0 {
		t.Fatalf("rejected registration should release user id, count=%d", count)
	}
	rereg := registerCommunityForReview(t, s, "rejectme")
	if rereg.Code != http.StatusAccepted {
		t.Fatalf("re-register after rejection status=%d body=%s", rereg.Code, rereg.Body.String())
	}
}

func TestBlogSiteOpenStatePermissionsAndAccessGate(t *testing.T) {
	s := newDLTestServer(t)
	userToken := seedDLUser(t, s.db, "u-blog", "blogger", "Blogger")
	now := nowISO()
	if _, err := s.db.Exec(`INSERT INTO blog_articles(id,author_id,title,summary,content_markdown,status,visibility,views,created_at,published_at) VALUES('art-blog','u-blog','Blog Gate','summary','body','published','public',3,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}

	blogToken := adminAPIToken(s, "blog_admin")
	siteToken := adminAPIToken(s, "site_admin")
	superToken := adminAPIToken(s, "superadmin")

	getDefault := adminAPIReq(s, http.MethodGet, "/api/admin/blog/site-state", blogToken, nil)
	if getDefault.Code != http.StatusOK {
		t.Fatalf("blog admin should read site state, status=%d body=%s", getDefault.Code, getDefault.Body.String())
	}
	var defaultBody struct {
		State BlogSiteState `json:"state"`
	}
	if err := json.Unmarshal(getDefault.Body.Bytes(), &defaultBody); err != nil {
		t.Fatal(err)
	}
	if !defaultBody.State.Open {
		t.Fatalf("default blog site state should be open")
	}

	siteGet := adminAPIReq(s, http.MethodGet, "/api/admin/blog/site-state", siteToken, nil)
	if siteGet.Code != http.StatusForbidden {
		t.Fatalf("site admin must not read blog switch, status=%d body=%s", siteGet.Code, siteGet.Body.String())
	}
	blogPut := adminAPIReq(s, http.MethodPut, "/api/admin/blog/site-state", blogToken, map[string]any{"open": false, "notice": "closed"})
	if blogPut.Code != http.StatusForbidden {
		t.Fatalf("blog admin must not update blog switch, status=%d body=%s", blogPut.Code, blogPut.Body.String())
	}
	sitePut := adminAPIReq(s, http.MethodPut, "/api/admin/blog/site-state", siteToken, map[string]any{"open": false, "notice": "closed"})
	if sitePut.Code != http.StatusForbidden {
		t.Fatalf("site admin must not update blog switch, status=%d body=%s", sitePut.Code, sitePut.Body.String())
	}

	closeResp := adminAPIReq(s, http.MethodPut, "/api/admin/blog/site-state", superToken, map[string]any{"open": false, "notice": "Closed for maintenance"})
	if closeResp.Code != http.StatusOK {
		t.Fatalf("superadmin close status=%d body=%s", closeResp.Code, closeResp.Body.String())
	}

	publicBlog := dlReq(s, http.MethodGet, "/blog", "", nil)
	if publicBlog.Code != http.StatusServiceUnavailable {
		t.Fatalf("closed blog page should be blocked, status=%d body=%s", publicBlog.Code, publicBlog.Body.String())
	}
	publicStaticBlog := dlReq(s, http.MethodGet, "/static/blog.html", "", nil)
	if publicStaticBlog.Code != http.StatusServiceUnavailable {
		t.Fatalf("direct static blog html should be blocked, status=%d body=%s", publicStaticBlog.Code, publicStaticBlog.Body.String())
	}
	publicAPI := dlReq(s, http.MethodGet, "/api/blog/recommendations", "", nil)
	if publicAPI.Code != http.StatusServiceUnavailable {
		t.Fatalf("closed public blog API should be blocked, status=%d body=%s", publicAPI.Code, publicAPI.Body.String())
	}
	userAPI := dlReq(s, http.MethodGet, "/api/blog/recommendations", userToken, nil)
	if userAPI.Code != http.StatusServiceUnavailable {
		t.Fatalf("closed logged-in user blog API should be blocked, status=%d body=%s", userAPI.Code, userAPI.Body.String())
	}
	adminBypassAPI := adminAPIReq(s, http.MethodGet, "/api/blog/recommendations", blogToken, nil)
	if adminBypassAPI.Code != http.StatusOK {
		t.Fatalf("blog admin should bypass API closure for inspection, status=%d body=%s", adminBypassAPI.Code, adminBypassAPI.Body.String())
	}

	var users, articles int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM community_users`).Scan(&users); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM blog_articles`).Scan(&articles); err != nil {
		t.Fatal(err)
	}
	if users != 1 || articles != 1 {
		t.Fatalf("closing blog must not delete cached data/users/articles, users=%d articles=%d", users, articles)
	}

	reopen := adminAPIReq(s, http.MethodPut, "/api/admin/blog/site-state", superToken, map[string]any{"open": true, "notice": "open"})
	if reopen.Code != http.StatusOK {
		t.Fatalf("superadmin reopen status=%d body=%s", reopen.Code, reopen.Body.String())
	}
	openAPI := dlReq(s, http.MethodGet, "/api/blog/recommendations", "", nil)
	if openAPI.Code != http.StatusOK {
		t.Fatalf("reopened public blog API should work, status=%d body=%s", openAPI.Code, openAPI.Body.String())
	}
}

func TestStaticBackupAndSecretFilesAreNotServed(t *testing.T) {
	s := newDLTestServer(t)
	staticDir := t.TempDir()
	s.cfg.StaticDir = staticDir
	if err := os.WriteFile(filepath.Join(staticDir, "ok.js"), []byte("console.log('ok')"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"app.js.codex_backup", ".env", "server.go", "flyteam.db", "trace.log", "file.tmp"} {
		if err := os.WriteFile(filepath.Join(staticDir, name), []byte("secret"), 0644); err != nil {
			t.Fatal(err)
		}
		rr := dlReq(s, http.MethodGet, "/static/"+name, "", nil)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("%s should not be served, status=%d body=%s", name, rr.Code, rr.Body.String())
		}
	}
	ok := dlReq(s, http.MethodGet, "/static/ok.js", "", nil)
	if ok.Code != http.StatusOK {
		t.Fatalf("normal static asset should still be served, status=%d body=%s", ok.Code, ok.Body.String())
	}
}
