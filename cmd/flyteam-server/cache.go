package main

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

type rateBucketCache struct {
	Entries []int64 `json:"entries"`
}

func parseCacheTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func cacheExpired(raw string) bool {
	if t, ok := parseCacheTime(raw); ok {
		return time.Now().UTC().After(t)
	}
	return false
}

func cacheExpiryValue(expiresAt time.Time) any {
	if expiresAt.IsZero() {
		return nil
	}
	return expiresAt.UTC().Format(time.RFC3339Nano)
}

func loadCacheJSONDB(db *sql.DB, scope, key string, dst any) bool {
	if db == nil || strings.TrimSpace(scope) == "" || strings.TrimSpace(key) == "" {
		return false
	}
	var raw string
	var expires sql.NullString
	if err := db.QueryRow(`SELECT value_json, expires_at FROM app_cache WHERE scope=? AND key=?`, scope, key).Scan(&raw, &expires); err != nil {
		return false
	}
	if expires.Valid && cacheExpired(expires.String) {
		_, _ = db.Exec(`DELETE FROM app_cache WHERE scope=? AND key=?`, scope, key)
		return false
	}
	return json.Unmarshal([]byte(raw), dst) == nil
}

func saveCacheJSONDB(db *sql.DB, scope, key string, data any, expiresAt time.Time) error {
	if db == nil {
		return sql.ErrConnDone
	}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO app_cache(scope,key,value_json,expires_at,updated_at) VALUES(?,?,?,?,?)
		ON CONFLICT(scope,key) DO UPDATE SET value_json=excluded.value_json, expires_at=excluded.expires_at, updated_at=excluded.updated_at`,
		scope, key, string(b), cacheExpiryValue(expiresAt), nowISO())
	return err
}

func deleteCacheDB(db *sql.DB, scope, key string) error {
	if db == nil {
		return sql.ErrConnDone
	}
	_, err := db.Exec(`DELETE FROM app_cache WHERE scope=? AND key=?`, scope, key)
	return err
}

func cleanupCacheDB(db *sql.DB, scope string) error {
	if db == nil {
		return sql.ErrConnDone
	}
	if strings.TrimSpace(scope) == "" {
		_, err := db.Exec(`DELETE FROM app_cache WHERE expires_at IS NOT NULL AND expires_at<>'' AND expires_at < ?`, nowISO())
		return err
	}
	_, err := db.Exec(`DELETE FROM app_cache WHERE scope=? AND expires_at IS NOT NULL AND expires_at<>'' AND expires_at < ?`, scope, nowISO())
	return err
}

func (s *Server) loadCacheJSON(scope, key string, dst any) bool {
	return loadCacheJSONDB(s.db, scope, key, dst)
}

func (s *Server) saveCacheJSON(scope, key string, data any, expiresAt time.Time) error {
	return saveCacheJSONDB(s.db, scope, key, data, expiresAt)
}

func (s *Server) deleteCache(scope, key string) {
	_ = deleteCacheDB(s.db, scope, key)
}

func (s *Server) cleanupCache(scope string) {
	_ = cleanupCacheDB(s.db, scope)
}

func (s *Server) checkRateLimitDB(key string, limit int, window time.Duration, consume bool) bool {
	if s.db == nil {
		return true
	}
	now := time.Now()
	cut := now.Add(-window).UnixNano()
	bucket := rateBucketCache{Entries: []int64{}}
	_ = s.loadCacheJSON("rate", key, &bucket)
	filtered := make([]int64, 0, len(bucket.Entries)+1)
	for _, ts := range bucket.Entries {
		if ts > cut {
			filtered = append(filtered, ts)
		}
	}
	changed := len(filtered) != len(bucket.Entries)
	if len(filtered) >= limit {
		if changed {
			_ = s.saveCacheJSON("rate", key, rateBucketCache{Entries: filtered}, now.Add(window+time.Minute))
		}
		return false
	}
	if consume {
		filtered = append(filtered, now.UnixNano())
		changed = true
	}
	if changed {
		_ = s.saveCacheJSON("rate", key, rateBucketCache{Entries: filtered}, now.Add(window+time.Minute))
	}
	return true
}
