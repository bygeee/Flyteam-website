package app

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
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

func TestGrandUserAccountLifecycle(t *testing.T) {
	s := newDLTestServer(t)

	register := dlReq(s, http.MethodPost, "/api/users/register", "", map[string]any{"nickname": "Grand", "user_id": "Grand_User", "password": "SafePass2026"})
	if register.Code != http.StatusAccepted {
		t.Fatalf("register status=%d body=%s", register.Code, register.Body.String())
	}
	var registeredID, registeredStatus string
	if err := s.db.QueryRow(`SELECT user_id,status FROM community_users WHERE user_id='grand_user'`).Scan(&registeredID, &registeredStatus); err != nil {
		t.Fatal(err)
	}
	if registeredID != "grand_user" || registeredStatus != "pending" {
		t.Fatalf("bad registered user id=%q status=%q", registeredID, registeredStatus)
	}

	pendingLogin := dlReq(s, http.MethodPost, "/api/users/login", "", map[string]any{"user_id": "grand_user", "password": "SafePass2026"})
	if pendingLogin.Code != http.StatusForbidden {
		t.Fatalf("pending login status=%d body=%s", pendingLogin.Code, pendingLogin.Body.String())
	}
	if _, err := s.db.Exec(`UPDATE community_users SET status='active' WHERE user_id='grand_user'`); err != nil {
		t.Fatal(err)
	}

	login := dlReq(s, http.MethodPost, "/api/users/login", "", map[string]any{"user_id": "grand_user", "password": "SafePass2026"})
	if login.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", login.Code, login.Body.String())
	}
	loginBody := decodeDLBody(t, login)
	token := loginBody["token"].(string)
	if token == "" {
		t.Fatal("expected login token")
	}

	me := dlReq(s, http.MethodGet, "/api/users/me", token, nil)
	if me.Code != http.StatusOK {
		t.Fatalf("me status=%d body=%s", me.Code, me.Body.String())
	}
	meUser := decodeDLBody(t, me)["user"].(map[string]any)
	if meUser["user_id"] != "grand_user" {
		t.Fatalf("bad me user: %#v", meUser)
	}

	settings := dlReq(s, http.MethodPut, "/api/users/me/settings", token, map[string]any{"nickname": "Grand New", "user_id": "grand_new", "bio": "负责社区后端"})
	if settings.Code != http.StatusOK {
		t.Fatalf("settings status=%d body=%s", settings.Code, settings.Body.String())
	}
	var nickname, userID, bio string
	if err := s.db.QueryRow(`SELECT nickname,user_id,bio FROM community_users WHERE user_id='grand_new'`).Scan(&nickname, &userID, &bio); err != nil {
		t.Fatal(err)
	}
	if nickname != "Grand New" || userID != "grand_new" || bio != "负责社区后端" {
		t.Fatalf("bad settings nickname=%q userID=%q bio=%q", nickname, userID, bio)
	}

	otherTok := seedDLUser(t, s.db, "u-other", "other", "Other")
	forbidden := dlReq(s, http.MethodPut, "/api/users/grand_new", otherTok, map[string]any{"nickname": "Hijack", "bio": "bad"})
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("foreign profile update status=%d body=%s", forbidden.Code, forbidden.Body.String())
	}

	wrongPassword := dlReq(s, http.MethodPut, "/api/users/me/password", token, map[string]any{"old_password": "WrongPass2026", "new_password": "BetterPass2026"})
	if wrongPassword.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password status=%d body=%s", wrongPassword.Code, wrongPassword.Body.String())
	}
	changePassword := dlReq(s, http.MethodPut, "/api/users/me/password", token, map[string]any{"old_password": "SafePass2026", "new_password": "BetterPass2026"})
	if changePassword.Code != http.StatusOK {
		t.Fatalf("change password status=%d body=%s", changePassword.Code, changePassword.Body.String())
	}

	oldLogin := dlReq(s, http.MethodPost, "/api/users/login", "", map[string]any{"user_id": "grand_new", "password": "SafePass2026"})
	if oldLogin.Code != http.StatusUnauthorized {
		t.Fatalf("old login status=%d body=%s", oldLogin.Code, oldLogin.Body.String())
	}
	newLogin := dlReq(s, http.MethodPost, "/api/users/login", "", map[string]any{"user_id": "grand_new", "password": "BetterPass2026"})
	if newLogin.Code != http.StatusOK {
		t.Fatalf("new login status=%d body=%s", newLogin.Code, newLogin.Body.String())
	}

	logout := dlReq(s, http.MethodPost, "/api/users/logout", token, nil)
	if logout.Code != http.StatusOK {
		t.Fatalf("logout status=%d body=%s", logout.Code, logout.Body.String())
	}
	afterLogout := dlReq(s, http.MethodGet, "/api/users/me", token, nil)
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("after logout status=%d body=%s", afterLogout.Code, afterLogout.Body.String())
	}
}

func TestGrandBlogArticleLifecycle(t *testing.T) {
	s := newDLTestServer(t)
	authorTok := seedDLUser(t, s.db, "u-author", "author", "Author")
	otherTok := seedDLUser(t, s.db, "u-other", "other", "Other")

	create := dlReq(s, http.MethodPost, "/api/blog/articles", authorTok, map[string]any{
		"title":            "  Go 后端安全  ",
		"summary":          "第一版摘要",
		"content_markdown": "# 草稿内容\n\nhello",
		"tags":             []string{"Go", "security", "go", " 后端 "},
		"category":         "backend",
		"language":         "go",
		"status":           "draft",
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	article := decodeDLBody(t, create)["article"].(map[string]any)
	articleID := article["id"].(string)
	if article["title"] != "Go 后端安全" || article["status"] != "draft" {
		t.Fatalf("bad created article: %#v", article)
	}

	publicList := dlReq(s, http.MethodGet, "/api/blog/articles", "", nil)
	if publicList.Code != http.StatusOK {
		t.Fatalf("public list status=%d body=%s", publicList.Code, publicList.Body.String())
	}
	if len(decodeDLBody(t, publicList)["articles"].([]any)) != 0 {
		t.Fatalf("draft should not be public: %s", publicList.Body.String())
	}
	publicDetail := dlReq(s, http.MethodGet, "/api/blog/articles/"+articleID, "", nil)
	if publicDetail.Code != http.StatusUnauthorized {
		t.Fatalf("public draft detail status=%d body=%s", publicDetail.Code, publicDetail.Body.String())
	}
	authorDetail := dlReq(s, http.MethodGet, "/api/blog/articles/"+articleID, authorTok, nil)
	if authorDetail.Code != http.StatusOK {
		t.Fatalf("author draft detail status=%d body=%s", authorDetail.Code, authorDetail.Body.String())
	}

	blockedUpdate := dlReq(s, http.MethodPut, "/api/blog/articles/"+articleID, otherTok, map[string]any{"title": "Hijack", "content_markdown": "bad"})
	if blockedUpdate.Code != http.StatusForbidden {
		t.Fatalf("foreign update status=%d body=%s", blockedUpdate.Code, blockedUpdate.Body.String())
	}

	update := dlReq(s, http.MethodPut, "/api/blog/articles/"+articleID, authorTok, map[string]any{
		"title":            "Go 后端安全进阶",
		"summary":          "第二版摘要",
		"content_markdown": "# 发布内容\n\nupdated",
		"tags":             []string{"Go", "安全"},
		"category":         "security",
		"language":         "go",
		"status":           "draft",
	})
	if update.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", update.Code, update.Body.String())
	}
	var versionsBeforePublish int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM blog_article_versions WHERE article_id=?`, articleID).Scan(&versionsBeforePublish); err != nil {
		t.Fatal(err)
	}
	if versionsBeforePublish != 2 {
		t.Fatalf("expected create and update versions, got %d", versionsBeforePublish)
	}

	publish := dlReq(s, http.MethodPost, "/api/blog/articles/"+articleID+"/publish", authorTok, nil)
	if publish.Code != http.StatusOK {
		t.Fatalf("publish status=%d body=%s", publish.Code, publish.Body.String())
	}
	publishedArticle := decodeDLBody(t, publish)["article"].(map[string]any)
	if publishedArticle["status"] != "published" || publishedArticle["published_at"] == "" {
		t.Fatalf("bad published article: %#v", publishedArticle)
	}

	list := dlReq(s, http.MethodGet, "/api/blog/articles", "", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}
	items := decodeDLBody(t, list)["articles"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["id"] != articleID {
		t.Fatalf("bad public list: %#v", items)
	}
	if _, ok := items[0].(map[string]any)["content_markdown"]; ok {
		t.Fatalf("list should not include full content: %#v", items[0])
	}

	detail := dlReq(s, http.MethodGet, "/api/blog/articles/"+articleID, authorTok, nil)
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status=%d body=%s", detail.Code, detail.Body.String())
	}
	detailArticle := decodeDLBody(t, detail)["article"].(map[string]any)
	if detailArticle["content_markdown"] != "# 发布内容\n\nupdated" {
		t.Fatalf("bad detail article: %#v", detailArticle)
	}

	view := dlReq(s, http.MethodPost, "/api/blog/articles/"+articleID+"/view", authorTok, nil)
	if view.Code != http.StatusOK {
		t.Fatalf("view status=%d body=%s", view.Code, view.Body.String())
	}
	if got := int(decodeDLBody(t, view)["views"].(float64)); got != 1 {
		t.Fatalf("expected first view count 1, got %d", got)
	}
	var storedViews int
	if err := s.db.QueryRow(`SELECT views FROM blog_articles WHERE id=?`, articleID).Scan(&storedViews); err != nil {
		t.Fatal(err)
	}
	if storedViews != 1 {
		t.Fatalf("stored views=%d", storedViews)
	}
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

func TestGrandCommentAndReactionEdges(t *testing.T) {
	s := newDLTestServer(t)
	authorTok := seedDLUser(t, s.db, "u-author", "author", "Author")
	commenterTok := seedDLUser(t, s.db, "u-commenter", "commenter", "Commenter")
	otherTok := seedDLUser(t, s.db, "u-other", "other", "Other")
	now := nowISO()
	if _, err := s.db.Exec(`INSERT INTO blog_articles(id,author_id,title,content_markdown,status,visibility,created_at,published_at) VALUES('edge-art','u-author','互动边界','正文','published','public',?,?)`, now, now); err != nil {
		t.Fatal(err)
	}

	comment := dlReq(s, http.MethodPost, "/api/blog/articles/edge-art/comments", commenterTok, map[string]any{"content": " 第一条评论 "})
	if comment.Code != http.StatusOK {
		t.Fatalf("comment status=%d body=%s", comment.Code, comment.Body.String())
	}
	commentID := decodeDLBody(t, comment)["comment"].(map[string]any)["id"].(string)
	if commentID == "" {
		t.Fatal("expected comment id")
	}

	blockedEdit := dlReq(s, http.MethodPut, "/api/blog/comments/"+commentID, otherTok, map[string]any{"content": "抢改"})
	if blockedEdit.Code != http.StatusForbidden {
		t.Fatalf("blocked edit status=%d body=%s", blockedEdit.Code, blockedEdit.Body.String())
	}
	edit := dlReq(s, http.MethodPut, "/api/blog/comments/"+commentID, commenterTok, map[string]any{"content": "更新后的评论"})
	if edit.Code != http.StatusOK {
		t.Fatalf("edit status=%d body=%s", edit.Code, edit.Body.String())
	}
	var content string
	if err := s.db.QueryRow(`SELECT content FROM blog_comments WHERE id=?`, commentID).Scan(&content); err != nil {
		t.Fatal(err)
	}
	if content != "更新后的评论" {
		t.Fatalf("bad comment content=%q", content)
	}

	for _, path := range []string{"/api/blog/articles/edge-art/like", "/api/blog/articles/edge-art/favorite"} {
		set := dlReq(s, http.MethodPost, path, commenterTok, nil)
		if set.Code != http.StatusOK {
			t.Fatalf("set reaction %s status=%d body=%s", path, set.Code, set.Body.String())
		}
	}
	var likes, favorites, comments int
	if err := s.db.QueryRow(`SELECT likes,favorites,comments FROM blog_articles WHERE id='edge-art'`).Scan(&likes, &favorites, &comments); err != nil {
		t.Fatal(err)
	}
	if likes != 1 || favorites != 1 || comments != 1 {
		t.Fatalf("bad counters after set likes=%d fav=%d comments=%d", likes, favorites, comments)
	}
	for _, path := range []string{"/api/blog/articles/edge-art/like", "/api/blog/articles/edge-art/favorite"} {
		unset := dlReq(s, http.MethodDelete, path, commenterTok, nil)
		if unset.Code != http.StatusOK {
			t.Fatalf("unset reaction %s status=%d body=%s", path, unset.Code, unset.Body.String())
		}
	}
	if err := s.db.QueryRow(`SELECT likes,favorites,comments FROM blog_articles WHERE id='edge-art'`).Scan(&likes, &favorites, &comments); err != nil {
		t.Fatal(err)
	}
	if likes != 0 || favorites != 0 || comments != 1 {
		t.Fatalf("bad counters after unset likes=%d fav=%d comments=%d", likes, favorites, comments)
	}

	blockedDelete := dlReq(s, http.MethodDelete, "/api/blog/comments/"+commentID, otherTok, nil)
	if blockedDelete.Code != http.StatusForbidden {
		t.Fatalf("blocked delete status=%d body=%s", blockedDelete.Code, blockedDelete.Body.String())
	}
	deleteComment := dlReq(s, http.MethodDelete, "/api/blog/comments/"+commentID, commenterTok, nil)
	if deleteComment.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", deleteComment.Code, deleteComment.Body.String())
	}
	if err := s.db.QueryRow(`SELECT likes,favorites,comments FROM blog_articles WHERE id='edge-art'`).Scan(&likes, &favorites, &comments); err != nil {
		t.Fatal(err)
	}
	if likes != 0 || favorites != 0 || comments != 0 {
		t.Fatalf("bad counters after delete likes=%d fav=%d comments=%d", likes, favorites, comments)
	}

	authorDelete := dlReq(s, http.MethodDelete, "/api/blog/comments/"+commentID, authorTok, nil)
	if authorDelete.Code != http.StatusNotFound {
		t.Fatalf("already deleted status=%d body=%s", authorDelete.Code, authorDelete.Body.String())
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
	if _, err := s.db.Exec(`INSERT INTO friendships(user_a,user_b,created_at) VALUES('u-a','u-b',?)`, nowISO()); err != nil {
		t.Fatal(err)
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

func TestGrandSocialMessagingGroupPermissions(t *testing.T) {
	s := newDLTestServer(t)
	aliceTok := seedDLUser(t, s.db, "u-alice", "alice", "Alice")
	bobTok := seedDLUser(t, s.db, "u-bob", "bob", "Bob")
	carolTok := seedDLUser(t, s.db, "u-carol", "carol", "Carol")

	noFriendConv := dlReq(s, http.MethodPost, "/api/messages/conversations", aliceTok, map[string]any{"target_user_id": "bob"})
	if noFriendConv.Code != http.StatusForbidden {
		t.Fatalf("non-friend conversation status=%d body=%s", noFriendConv.Code, noFriendConv.Body.String())
	}

	friendReq := dlReq(s, http.MethodPost, "/api/friends/requests", aliceTok, map[string]any{"target_user_id": "bob", "message": "一起学习"})
	if friendReq.Code != http.StatusCreated {
		t.Fatalf("friend request status=%d body=%s", friendReq.Code, friendReq.Body.String())
	}
	requestID := decodeDLBody(t, friendReq)["request"].(map[string]any)["id"].(string)
	wrongAccept := dlReq(s, http.MethodPost, "/api/friends/requests/"+requestID+"/accept", aliceTok, nil)
	if wrongAccept.Code != http.StatusForbidden {
		t.Fatalf("wrong accept status=%d body=%s", wrongAccept.Code, wrongAccept.Body.String())
	}
	accept := dlReq(s, http.MethodPost, "/api/friends/requests/"+requestID+"/accept", bobTok, nil)
	if accept.Code != http.StatusOK {
		t.Fatalf("accept status=%d body=%s", accept.Code, accept.Body.String())
	}
	if !s.areFriends("u-alice", "u-bob") {
		t.Fatal("expected alice and bob to become friends")
	}

	conv := dlReq(s, http.MethodPost, "/api/messages/conversations", aliceTok, map[string]any{"target_user_id": "bob"})
	if conv.Code != http.StatusOK {
		t.Fatalf("conversation status=%d body=%s", conv.Code, conv.Body.String())
	}
	convID := decodeDLBody(t, conv)["conversation"].(map[string]any)["id"].(string)
	thirdRead := dlReq(s, http.MethodGet, "/api/messages/conversations/"+convID, carolTok, nil)
	if thirdRead.Code != http.StatusNotFound {
		t.Fatalf("third read status=%d body=%s", thirdRead.Code, thirdRead.Body.String())
	}
	thirdSend := dlReq(s, http.MethodPost, "/api/messages/conversations/"+convID+"/messages", carolTok, map[string]any{"content": "插话"})
	if thirdSend.Code != http.StatusNotFound {
		t.Fatalf("third send status=%d body=%s", thirdSend.Code, thirdSend.Body.String())
	}
	send := dlReq(s, http.MethodPost, "/api/messages/conversations/"+convID+"/messages", aliceTok, map[string]any{"content": "hello bob"})
	if send.Code != http.StatusOK {
		t.Fatalf("send status=%d body=%s", send.Code, send.Body.String())
	}
	bobMessages := dlReq(s, http.MethodGet, "/api/messages/conversations/"+convID+"/messages", bobTok, nil)
	if bobMessages.Code != http.StatusOK || len(decodeDLBody(t, bobMessages)["items"].([]any)) != 1 {
		t.Fatalf("bob messages status=%d body=%s", bobMessages.Code, bobMessages.Body.String())
	}

	badGroup := dlReq(s, http.MethodPost, "/api/groups", aliceTok, map[string]any{"name": "私有安全组", "visibility": "private", "member_user_ids": []string{"carol"}})
	if badGroup.Code != http.StatusForbidden {
		t.Fatalf("invite non-friend status=%d body=%s", badGroup.Code, badGroup.Body.String())
	}
	group := dlReq(s, http.MethodPost, "/api/groups", aliceTok, map[string]any{"name": "私有安全组", "visibility": "private", "member_user_ids": []string{"bob"}})
	if group.Code != http.StatusOK {
		t.Fatalf("group status=%d body=%s", group.Code, group.Body.String())
	}
	groupID := decodeDLBody(t, group)["group"].(map[string]any)["id"].(string)
	carolView := dlReq(s, http.MethodGet, "/api/groups/"+groupID, carolTok, nil)
	if carolView.Code != http.StatusForbidden {
		t.Fatalf("private group view status=%d body=%s", carolView.Code, carolView.Body.String())
	}
	carolJoin := dlReq(s, http.MethodPost, "/api/groups/"+groupID+"/members", carolTok, map[string]any{})
	if carolJoin.Code != http.StatusForbidden {
		t.Fatalf("private group join status=%d body=%s", carolJoin.Code, carolJoin.Body.String())
	}
	nonOwnerKick := dlReq(s, http.MethodDelete, "/api/groups/"+groupID+"/members/alice", bobTok, nil)
	if nonOwnerKick.Code != http.StatusForbidden {
		t.Fatalf("non-owner kick status=%d body=%s", nonOwnerKick.Code, nonOwnerKick.Body.String())
	}
	nonOwnerUpdate := dlReq(s, http.MethodPut, "/api/groups/"+groupID, bobTok, map[string]any{"name": "改名", "visibility": "private"})
	if nonOwnerUpdate.Code != http.StatusForbidden {
		t.Fatalf("non-owner update status=%d body=%s", nonOwnerUpdate.Code, nonOwnerUpdate.Body.String())
	}
	bobSend := dlReq(s, http.MethodPost, "/api/groups/"+groupID+"/messages", bobTok, map[string]any{"content": "hello group"})
	if bobSend.Code != http.StatusOK {
		t.Fatalf("bob group send status=%d body=%s", bobSend.Code, bobSend.Body.String())
	}
	kickBob := dlReq(s, http.MethodDelete, "/api/groups/"+groupID+"/members/bob", aliceTok, nil)
	if kickBob.Code != http.StatusOK {
		t.Fatalf("owner kick status=%d body=%s", kickBob.Code, kickBob.Body.String())
	}
	bobReadAfterKick := dlReq(s, http.MethodGet, "/api/groups/"+groupID+"/messages", bobTok, nil)
	if bobReadAfterKick.Code != http.StatusForbidden {
		t.Fatalf("kicked read status=%d body=%s", bobReadAfterKick.Code, bobReadAfterKick.Body.String())
	}
	bobSendAfterKick := dlReq(s, http.MethodPost, "/api/groups/"+groupID+"/messages", bobTok, map[string]any{"content": "after kick"})
	if bobSendAfterKick.Code != http.StatusForbidden {
		t.Fatalf("kicked send status=%d body=%s", bobSendAfterKick.Code, bobSendAfterKick.Body.String())
	}
}

func TestGrandRAGChatStream(t *testing.T) {
	llm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/embeddings":
			writeJSON(w, http.StatusOK, map[string]any{"data": []map[string]any{{"index": 0, "embedding": []float64{1, 0}}}})
		case "/chat/completions":
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode llm request: %v", err)
			}
			if req["stream"] != true {
				t.Fatalf("expected stream request, got %#v", req)
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Flyteam\"}}]}\n\n"))
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" 是安全团队\"}}]}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer llm.Close()

	s := newDLTestServer(t)
	s.cfg.OpenAIAPIKey = "test-key"
	s.cfg.OpenAIBaseURL = llm.URL
	s.cfg.EmbeddingModel = "embedding-test"
	s.cfg.ChatModel = "chat-test"
	s.cfg.RetrievalMinRelevance = 0.01
	s.rag = NewRagService(s.cfg, s.db)
	s.rag.Index.Chunks = []RagChunk{{ID: "chunk1", Source: "flyteam.pdf", Page: 3, Text: "Flyteam 是安全团队，负责竞赛、招新和技术分享。", Embedding: []float64{1, 0}}}

	rr := dlReq(s, http.MethodPost, "/api/chat/stream", "", map[string]any{"question": "Flyteam 是什么？", "top_k": 1})
	if rr.Code != http.StatusOK {
		t.Fatalf("stream status=%d body=%s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("expected sse content type, got %q", ct)
	}
	body := rr.Body.String()
	for _, want := range []string{"event: token", "Flyteam", " 是安全团队", "event: sources", "flyteam.pdf", "event: done"} {
		if !strings.Contains(body, want) {
			t.Fatalf("stream body missing %q: %s", want, body)
		}
	}
}

func TestGrandRAGChatStreamFallback(t *testing.T) {
	s := newDLTestServer(t)
	s.cfg.OpenAIAPIKey = "test-key"
	s.cfg.OpenAIBaseURL = "http://127.0.0.1:1"
	s.rag = NewRagService(s.cfg, s.db)

	rr := dlReq(s, http.MethodPost, "/api/chat/stream", "", map[string]any{"question": "没有资料的问题"})
	if rr.Code != http.StatusOK {
		t.Fatalf("fallback status=%d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, want := range []string{"event: token", noInfoAnswer, "event: sources", "event: done"} {
		if !strings.Contains(body, want) {
			t.Fatalf("fallback body missing %q: %s", want, body)
		}
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
