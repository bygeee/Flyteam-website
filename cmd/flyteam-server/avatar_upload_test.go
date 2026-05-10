package main

import (
	"bytes"
	"encoding/hex"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testAvatarPNG(t *testing.T) []byte {
	t.Helper()
	b, err := hex.DecodeString("89504e470d0a1a0a0000000d49484452000000010000000108060000001f15c4890000000a49444154789c63000100000500010d0a2db40000000049454e44ae426082")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func uploadAvatarReq(t *testing.T, s *Server, token, filename string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("files", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(testAvatarPNG(t)); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/upload/avatar", &buf)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-User-Token", token)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	return rr
}

func TestUploadAvatarReplacesPreviousAvatar(t *testing.T) {
	s := newDLTestServer(t)
	s.cfg.AvatarUploadDir = t.TempDir()
	s.cfg.MaxImageUploadBytes = 1024 * 1024
	token := seedDLUser(t, s.db, "u-avatar", "avataruser", "Avatar User")

	first := uploadAvatarReq(t, s, token, "first.png")
	if first.Code != http.StatusOK {
		t.Fatalf("first avatar upload status=%d body=%s", first.Code, first.Body.String())
	}
	firstURL, _ := decodeDLBody(t, first)["avatar_url"].(string)
	if !strings.HasPrefix(firstURL, "/uploads/avatars/") {
		t.Fatalf("bad first avatar url %q", firstURL)
	}
	firstPath := filepath.Join(s.cfg.AvatarUploadDir, filepath.Base(firstURL))
	if _, err := os.Stat(firstPath); err != nil {
		t.Fatalf("first avatar should exist: %v", err)
	}

	second := uploadAvatarReq(t, s, token, "second.png")
	if second.Code != http.StatusOK {
		t.Fatalf("second avatar upload status=%d body=%s", second.Code, second.Body.String())
	}
	secondURL, _ := decodeDLBody(t, second)["avatar_url"].(string)
	if secondURL == "" || secondURL == firstURL {
		t.Fatalf("second avatar should get a new url, first=%q second=%q", firstURL, secondURL)
	}
	var stored string
	if err := s.db.QueryRow(`SELECT COALESCE(avatar_url,'') FROM community_users WHERE id='u-avatar'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != secondURL {
		t.Fatalf("database avatar_url=%q, want %q", stored, secondURL)
	}
	if _, err := os.Stat(firstPath); !os.IsNotExist(err) {
		t.Fatalf("old avatar file should be removed, stat err=%v", err)
	}
	secondPath := filepath.Join(s.cfg.AvatarUploadDir, filepath.Base(secondURL))
	if _, err := os.Stat(secondPath); err != nil {
		t.Fatalf("second avatar should exist: %v", err)
	}
}
