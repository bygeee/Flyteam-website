package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommunityFrontendAuthGate(t *testing.T) {
	s := newDLTestServer(t)
	s.cfg.StaticDir = filepath.Join("..", "..", "app", "static")

	for _, path := range []string{"/blog", "/user-login", "/user-register"} {
		rr := dlReq(s, http.MethodGet, path, "", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("guest should access %s, status=%d body=%s", path, rr.Code, rr.Body.String())
		}
	}

	privatePaths := []string{
		"/blog/art-1",
		"/editor",
		"/account",
		"/messages",
		"/groups",
		"/space/alice",
		"/static/article.html",
		"/static/editor.html",
		"/static/account.html",
		"/static/messages.html",
		"/static/groups.html",
		"/static/space.html",
	}
	for _, path := range privatePaths {
		rr := dlReq(s, http.MethodGet, path, "", nil)
		if rr.Code != http.StatusFound {
			t.Fatalf("guest should be redirected for %s, status=%d body=%s", path, rr.Code, rr.Body.String())
		}
		if loc := rr.Header().Get("Location"); !strings.HasPrefix(loc, "/user-login?next=") {
			t.Fatalf("guest redirect for %s should point to user-login, got %q", path, loc)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/editor", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("JSON guest should receive 401, status=%d body=%s", rr.Code, rr.Body.String())
	}

	privateAPIPaths := []string{"/api/blog/articles/art-1", "/api/blog/articles/art-1/comments", "/api/users/alice", "/api/groups", "/api/messages/conversations"}
	for _, path := range privateAPIPaths {
		rr := dlReq(s, http.MethodGet, path, "", nil)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("guest should not access private community API %s, status=%d body=%s", path, rr.Code, rr.Body.String())
		}
	}

	token := seedDLUser(t, s.db, "u-auth", "alice", "Alice")
	apiProfile := dlReq(s, http.MethodGet, "/api/users/alice", "", nil)
	if apiProfile.Code != http.StatusUnauthorized {
		t.Fatalf("guest should not read profile API, status=%d body=%s", apiProfile.Code, apiProfile.Body.String())
	}
	guestSearch := dlReq(s, http.MethodGet, "/api/search?q=alice", "", nil)
	if guestSearch.Code != http.StatusOK {
		t.Fatalf("guest blog search should still work for articles, status=%d body=%s", guestSearch.Code, guestSearch.Body.String())
	}
	if users := decodeDLBody(t, guestSearch)["users"].([]any); len(users) != 0 {
		t.Fatalf("guest search should not expose user results, got %#v", users)
	}
	loggedSearch := dlReq(s, http.MethodGet, "/api/search?q=alice", token, nil)
	if loggedSearch.Code != http.StatusOK {
		t.Fatalf("logged search status=%d body=%s", loggedSearch.Code, loggedSearch.Body.String())
	}
	if users := decodeDLBody(t, loggedSearch)["users"].([]any); len(users) == 0 {
		t.Fatalf("logged search should include user results")
	}
	loggedProfile := dlReq(s, http.MethodGet, "/api/users/alice", token, nil)
	if loggedProfile.Code != http.StatusOK {
		t.Fatalf("logged user should read profile API, status=%d body=%s", loggedProfile.Code, loggedProfile.Body.String())
	}

	for _, path := range []string{"/blog/art-1", "/editor", "/account", "/messages", "/groups", "/space/alice"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(&http.Cookie{Name: "user_session", Value: token})
		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("logged-in browser cookie should access %s, status=%d body=%s", path, rr.Code, rr.Body.String())
		}
	}

	if _, err := s.db.Exec(`UPDATE community_users SET status='banned' WHERE id='u-auth'`); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodGet, "/editor", nil)
	req.AddCookie(&http.Cookie{Name: "user_session", Value: token})
	banned := httptest.NewRecorder()
	s.ServeHTTP(banned, req)
	if banned.Code != http.StatusFound {
		t.Fatalf("banned user should be redirected from private page, status=%d body=%s", banned.Code, banned.Body.String())
	}
}
