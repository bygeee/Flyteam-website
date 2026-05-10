package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

func pathValue(path, prefix string) string {
	return strings.Trim(strings.TrimPrefix(path, prefix), "/")
}

func isMutating(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}

func wantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json") || strings.HasPrefix(r.URL.Path, "/api/")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]any{"detail": detail})
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 2<<20))
	return dec.Decode(dst)
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func randomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return strings.TrimRight(strings.NewReplacer("+", "-", "/", "_").Replace(hex.EncodeToString(b)), "=")
}

func nowISO() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func writeJSONAtomic(path string, data any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if host == "127.0.0.1" || host == "::1" || host == "localhost" {
		if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
			first := strings.TrimSpace(strings.Split(xf, ",")[0])
			if len(first) >= 3 && len(first) <= 45 {
				return first
			}
		}
	}
	if host == "" {
		return "unknown"
	}
	return host
}

func (s *Server) checkRateLimit(key string, limit int, window time.Duration, consume bool) bool {
	if s.db != nil {
		return s.checkRateLimitDB(key, limit, window, consume)
	}
	s.rateMu.Lock()
	defer s.rateMu.Unlock()
	now := time.Now()
	cut := now.Add(-window)
	bucket := s.rate[key]
	out := bucket[:0]
	for _, t := range bucket {
		if t.After(cut) {
			out = append(out, t)
		}
	}
	bucket = out
	if len(bucket) >= limit {
		s.rate[key] = bucket
		return false
	}
	if consume {
		bucket = append(bucket, now)
	}
	s.rate[key] = bucket
	return true
}

func (s *Server) clearRateLimit(key string) {
	if s.db != nil {
		s.deleteCache("rate", key)
		return
	}
	s.rateMu.Lock()
	defer s.rateMu.Unlock()
	delete(s.rate, key)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func validHall(h string) string {
	switch h {
	case "binary", "web", "dev", "management":
		return h
	}
	return "binary"
}

func requireFields(ok bool, msg string) error {
	if !ok {
		return errors.New(msg)
	}
	return nil
}

func fatalIf(err error) {
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}
}
