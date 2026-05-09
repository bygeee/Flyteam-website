package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func newDLTestServer(t *testing.T) *Server {
	t.Helper()
	cfg := Config{DatabaseFile: filepath.Join(t.TempDir(), "flyteam-test.db")}
	db, err := openDatabase(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return &Server{cfg: cfg, db: db, sessions: map[string]AdminSession{}, rate: map[string][]time.Time{}, captchas: map[string]CaptchaEntry{}}
}

func seedDLUser(t *testing.T, db *sql.DB, pk, userID, nickname string) string {
	t.Helper()
	now := nowISO()
	_, err := db.Exec(`INSERT INTO community_users(id,user_id,nickname,password_hash,salt,role,status,created_at) VALUES(?,?,?,?,?,'user','active',?)`, pk, userID, nickname, "hash", "salt", now)
	if err != nil {
		t.Fatal(err)
	}
	token := "token-" + userID
	_, err = db.Exec(`INSERT INTO community_sessions(session_token,user_pk,csrf_token,expires_at,created_at) VALUES(?,?,?,?,?)`, token, pk, "csrf-"+userID, time.Now().UTC().Add(time.Hour).Format(time.RFC3339Nano), now)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func dlReq(s *Server, method, path, token string, body any) *httptest.ResponseRecorder {
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
		req.Header.Set("X-User-Token", token)
	}
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	return rr
}

func decodeDLBody(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body: %v body=%s", err, rr.Body.String())
	}
	return out
}

func TestDLCommentsReactionsAndNotifications(t *testing.T) {
	s := newDLTestServer(t)
	aTok := seedDLUser(t, s.db, "u-a", "alice", "Alice")
	bTok := seedDLUser(t, s.db, "u-b", "bob", "Bob")
	now := nowISO()
	_, err := s.db.Exec(`INSERT INTO blog_articles(id,author_id,title,content_markdown,status,visibility,views,created_at,published_at) VALUES('art1','u-a','Go 安全','正文','published','public',7,?,?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}

	unauth := dlReq(s, http.MethodPost, "/api/blog/articles/art1/comments", "", map[string]any{"content": "hello"})
	if unauth.Code != http.StatusUnauthorized {
		t.Fatalf("unauth comment status=%d body=%s", unauth.Code, unauth.Body.String())
	}

	comment := dlReq(s, http.MethodPost, "/api/blog/articles/art1/comments", bTok, map[string]any{"content": "写得好"})
	if comment.Code != http.StatusOK {
		t.Fatalf("comment status=%d body=%s", comment.Code, comment.Body.String())
	}
	like := dlReq(s, http.MethodPost, "/api/blog/articles/art1/like", bTok, nil)
	if like.Code != http.StatusOK {
		t.Fatalf("like status=%d body=%s", like.Code, like.Body.String())
	}
	likeAgain := dlReq(s, http.MethodPost, "/api/blog/articles/art1/like", bTok, nil)
	if likeAgain.Code != http.StatusOK {
		t.Fatalf("like again status=%d body=%s", likeAgain.Code, likeAgain.Body.String())
	}
	fav := dlReq(s, http.MethodPost, "/api/blog/articles/art1/favorite", bTok, nil)
	if fav.Code != http.StatusOK {
		t.Fatalf("fav status=%d body=%s", fav.Code, fav.Body.String())
	}

	var likes, favorites, comments int
	if err := s.db.QueryRow(`SELECT likes,favorites,comments FROM blog_articles WHERE id='art1'`).Scan(&likes, &favorites, &comments); err != nil {
		t.Fatal(err)
	}
	if likes != 1 || favorites != 1 || comments != 1 {
		t.Fatalf("bad stats likes=%d fav=%d comments=%d", likes, favorites, comments)
	}

	notify := dlReq(s, http.MethodGet, "/api/notifications", aTok, nil)
	if notify.Code != http.StatusOK {
		t.Fatalf("notify status=%d body=%s", notify.Code, notify.Body.String())
	}
	body := decodeDLBody(t, notify)
	if int(body["unread_count"].(float64)) < 1 {
		t.Fatalf("expected unread notification, got %#v", body)
	}
}

func TestDLFollowMessagesAndGroups(t *testing.T) {
	s := newDLTestServer(t)
	aTok := seedDLUser(t, s.db, "u-a", "alice", "Alice")
	bTok := seedDLUser(t, s.db, "u-b", "bob", "Bob")
	cTok := seedDLUser(t, s.db, "u-c", "carol", "Carol")

	self := dlReq(s, http.MethodPost, "/api/social/follows/alice", aTok, nil)
	if self.Code != http.StatusBadRequest {
		t.Fatalf("self follow status=%d", self.Code)
	}
	follow := dlReq(s, http.MethodPost, "/api/social/follows/bob", aTok, nil)
	if follow.Code != http.StatusOK {
		t.Fatalf("follow status=%d body=%s", follow.Code, follow.Body.String())
	}
	followAgain := dlReq(s, http.MethodPost, "/api/social/follows/bob", aTok, nil)
	if followAgain.Code != http.StatusOK {
		t.Fatalf("follow again status=%d", followAgain.Code)
	}
	var follows int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM social_follows WHERE follower_id='u-a' AND following_id='u-b'`).Scan(&follows)
	if follows != 1 {
		t.Fatalf("duplicate follows=%d", follows)
	}

	conv := dlReq(s, http.MethodPost, "/api/messages/conversations", aTok, map[string]any{"target_user_id": "bob"})
	if conv.Code != http.StatusOK {
		t.Fatalf("conv status=%d body=%s", conv.Code, conv.Body.String())
	}
	convID := decodeDLBody(t, conv)["conversation"].(map[string]any)["id"].(string)
	msg := dlReq(s, http.MethodPost, "/api/messages/conversations/"+convID+"/messages", aTok, map[string]any{"content": "hi"})
	if msg.Code != http.StatusOK {
		t.Fatalf("msg status=%d body=%s", msg.Code, msg.Body.String())
	}
	blocked := dlReq(s, http.MethodGet, "/api/messages/conversations/"+convID+"/messages", cTok, nil)
	if blocked.Code != http.StatusNotFound {
		t.Fatalf("third party status=%d body=%s", blocked.Code, blocked.Body.String())
	}
	read := dlReq(s, http.MethodGet, "/api/messages/conversations/"+convID+"/messages", bTok, nil)
	if read.Code != http.StatusOK {
		t.Fatalf("read status=%d body=%s", read.Code, read.Body.String())
	}

	group := dlReq(s, http.MethodPost, "/api/groups", aTok, map[string]any{"name": "安全学习"})
	if group.Code != http.StatusOK {
		t.Fatalf("group status=%d body=%s", group.Code, group.Body.String())
	}
	groupID := decodeDLBody(t, group)["group"].(map[string]any)["id"].(string)
	join := dlReq(s, http.MethodPost, "/api/groups/"+groupID+"/members", bTok, map[string]any{})
	if join.Code != http.StatusOK {
		t.Fatalf("join status=%d body=%s", join.Code, join.Body.String())
	}
	gm := dlReq(s, http.MethodPost, "/api/groups/"+groupID+"/messages", bTok, map[string]any{"content": "大家好"})
	if gm.Code != http.StatusOK {
		t.Fatalf("group msg status=%d body=%s", gm.Code, gm.Body.String())
	}
	kick := dlReq(s, http.MethodDelete, "/api/groups/"+groupID+"/members/bob", aTok, nil)
	if kick.Code != http.StatusOK {
		t.Fatalf("kick status=%d body=%s", kick.Code, kick.Body.String())
	}
	blockedMsg := dlReq(s, http.MethodPost, "/api/groups/"+groupID+"/messages", bTok, map[string]any{"content": "还能说吗"})
	if blockedMsg.Code != http.StatusForbidden {
		t.Fatalf("kicked msg status=%d body=%s", blockedMsg.Code, blockedMsg.Body.String())
	}
}

func TestDLSearchAndRecommendations(t *testing.T) {
	s := newDLTestServer(t)
	seedDLUser(t, s.db, "u-a", "alice", "Alice")
	now := nowISO()
	_, err := s.db.Exec(`INSERT INTO blog_articles(id,author_id,title,summary,content_markdown,status,visibility,views,likes,favorites,comments,created_at,published_at) VALUES
		('low','u-a','低热度 Web','Web 摘要','web','published','public',1,0,0,0,?,?),
		('hot','u-a','高热度 Go 安全','Go 摘要','golang security','published','public',10,2,1,3,?,?)`, now, now, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = s.db.Exec(`INSERT INTO blog_article_tags(article_id,tag) VALUES('hot','go')`)
	search := dlReq(s, http.MethodGet, "/api/search?q=Go", "", nil)
	if search.Code != http.StatusOK {
		t.Fatalf("search status=%d body=%s", search.Code, search.Body.String())
	}
	if len(decodeDLBody(t, search)["articles"].([]any)) == 0 {
		t.Fatal("expected search articles")
	}
	rec := dlReq(s, http.MethodGet, "/api/blog/recommendations?limit=2", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("rec status=%d body=%s", rec.Code, rec.Body.String())
	}
	items := decodeDLBody(t, rec)["items"].([]any)
	if len(items) == 0 || items[0].(map[string]any)["id"] != "hot" {
		t.Fatalf("bad recommendations: %#v", items)
	}
}
