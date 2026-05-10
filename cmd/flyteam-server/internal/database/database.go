package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

func Open(databaseFile string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(databaseFile), 0755); err != nil {
		return nil, err
	}
	dsn := databaseFile
	if !strings.Contains(dsn, "?") {
		dsn += "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := InitSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func InitSchema(db *sql.DB) error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS app_kv (
			key TEXT PRIMARY KEY,
			value_json TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS app_cache (
			scope TEXT NOT NULL,
			key TEXT NOT NULL,
			value_json TEXT NOT NULL,
			expires_at TEXT,
			updated_at TEXT NOT NULL,
			PRIMARY KEY(scope, key)
		)`,
		`CREATE TABLE IF NOT EXISTS admin_users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			display_name TEXT,
			role TEXT NOT NULL DEFAULT 'admin',
			salt TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TEXT NOT NULL,
			last_login_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS community_users (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL UNIQUE,
			nickname TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			salt TEXT NOT NULL,
			avatar_url TEXT,
			bio TEXT,
			role TEXT NOT NULL DEFAULT 'user',
			status TEXT NOT NULL DEFAULT 'active',
			created_at TEXT NOT NULL,
			updated_at TEXT,
			last_login_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS community_sessions (
			session_token TEXT PRIMARY KEY,
			user_pk TEXT NOT NULL,
			csrf_token TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL,
			user_agent_hash TEXT,
			ip_hash TEXT,
			FOREIGN KEY(user_pk) REFERENCES community_users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS blog_articles (
			id TEXT PRIMARY KEY,
			author_id TEXT NOT NULL,
			title TEXT NOT NULL,
			slug TEXT,
			summary TEXT,
			cover_url TEXT,
			content_markdown TEXT NOT NULL,
			content_html TEXT,
			status TEXT NOT NULL DEFAULT 'draft',
			visibility TEXT NOT NULL DEFAULT 'public',
			language TEXT,
			category TEXT,
			pinned INTEGER NOT NULL DEFAULT 0,
			recommend_weight INTEGER NOT NULL DEFAULT 0,
			views INTEGER NOT NULL DEFAULT 0,
			likes INTEGER NOT NULL DEFAULT 0,
			favorites INTEGER NOT NULL DEFAULT 0,
			comments INTEGER NOT NULL DEFAULT 0,
			published_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT,
			FOREIGN KEY(author_id) REFERENCES community_users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS blog_article_tags (
			article_id TEXT NOT NULL,
			tag TEXT NOT NULL,
			PRIMARY KEY(article_id, tag),
			FOREIGN KEY(article_id) REFERENCES blog_articles(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS blog_article_versions (
			id TEXT PRIMARY KEY,
			article_id TEXT NOT NULL,
			title TEXT,
			summary TEXT,
			content_markdown TEXT,
			created_at TEXT NOT NULL,
			created_by TEXT NOT NULL,
			FOREIGN KEY(article_id) REFERENCES blog_articles(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS blog_comments (
			id TEXT PRIMARY KEY,
			article_id TEXT NOT NULL,
			author_id TEXT NOT NULL,
			parent_id TEXT,
			content TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'visible',
			created_at TEXT NOT NULL,
			updated_at TEXT,
			FOREIGN KEY(article_id) REFERENCES blog_articles(id) ON DELETE CASCADE,
			FOREIGN KEY(author_id) REFERENCES community_users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS blog_likes (
			article_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY(article_id, user_id),
			FOREIGN KEY(article_id) REFERENCES blog_articles(id) ON DELETE CASCADE,
			FOREIGN KEY(user_id) REFERENCES community_users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS blog_favorites (
			article_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY(article_id, user_id),
			FOREIGN KEY(article_id) REFERENCES blog_articles(id) ON DELETE CASCADE,
			FOREIGN KEY(user_id) REFERENCES community_users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS social_follows (
			follower_id TEXT NOT NULL,
			following_id TEXT NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY(follower_id, following_id),
			FOREIGN KEY(follower_id) REFERENCES community_users(id) ON DELETE CASCADE,
			FOREIGN KEY(following_id) REFERENCES community_users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS friend_requests (
			id TEXT PRIMARY KEY,
			requester_id TEXT NOT NULL,
			addressee_id TEXT NOT NULL,
			message TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			created_at TEXT NOT NULL,
			updated_at TEXT,
			FOREIGN KEY(requester_id) REFERENCES community_users(id) ON DELETE CASCADE,
			FOREIGN KEY(addressee_id) REFERENCES community_users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS friendships (
			user_a TEXT NOT NULL,
			user_b TEXT NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY(user_a, user_b),
			FOREIGN KEY(user_a) REFERENCES community_users(id) ON DELETE CASCADE,
			FOREIGN KEY(user_b) REFERENCES community_users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS private_conversations (
			id TEXT PRIMARY KEY,
			user_a TEXT NOT NULL,
			user_b TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT,
			last_message_at TEXT,
			UNIQUE(user_a, user_b),
			FOREIGN KEY(user_a) REFERENCES community_users(id) ON DELETE CASCADE,
			FOREIGN KEY(user_b) REFERENCES community_users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS private_messages (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			sender_id TEXT NOT NULL,
			content TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'normal',
			created_at TEXT NOT NULL,
			read_at TEXT,
			FOREIGN KEY(conversation_id) REFERENCES private_conversations(id) ON DELETE CASCADE,
			FOREIGN KEY(sender_id) REFERENCES community_users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS chat_groups (
			id TEXT PRIMARY KEY,
			owner_id TEXT NOT NULL,
			name TEXT NOT NULL,
			avatar_url TEXT,
			intro TEXT,
			visibility TEXT NOT NULL DEFAULT 'public',
			created_at TEXT NOT NULL,
			updated_at TEXT,
			FOREIGN KEY(owner_id) REFERENCES community_users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS chat_group_members (
			group_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'member',
			status TEXT NOT NULL DEFAULT 'active',
			joined_at TEXT NOT NULL,
			PRIMARY KEY(group_id, user_id),
			FOREIGN KEY(group_id) REFERENCES chat_groups(id) ON DELETE CASCADE,
			FOREIGN KEY(user_id) REFERENCES community_users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS chat_group_messages (
			id TEXT PRIMARY KEY,
			group_id TEXT NOT NULL,
			sender_id TEXT NOT NULL,
			content TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'normal',
			created_at TEXT NOT NULL,
			FOREIGN KEY(group_id) REFERENCES chat_groups(id) ON DELETE CASCADE,
			FOREIGN KEY(sender_id) REFERENCES community_users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS notifications (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			type TEXT NOT NULL,
			payload_json TEXT,
			read_at TEXT,
			created_at TEXT NOT NULL,
			FOREIGN KEY(user_id) REFERENCES community_users(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_blog_articles_public ON blog_articles(status, visibility, published_at)`,
		`CREATE INDEX IF NOT EXISTS idx_blog_articles_author ON blog_articles(author_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_blog_comments_article ON blog_comments(article_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_friend_requests_inbox ON friend_requests(addressee_id, status, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_friend_requests_outbox ON friend_requests(requester_id, status, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_friendships_user_b ON friendships(user_b, user_a)`,
		`CREATE INDEX IF NOT EXISTS idx_private_messages_conversation ON private_messages(conversation_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_group_messages_group ON chat_group_messages(group_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_app_cache_expiry ON app_cache(scope, expires_at)`,
	}
	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("schema failed: %w\n%s", err, stmt)
		}
	}
	return nil
}

func LoadJSON(db *sql.DB, key string, dst any) bool {
	if db == nil {
		return false
	}
	var raw string
	if err := db.QueryRow(`SELECT value_json FROM app_kv WHERE key=?`, key).Scan(&raw); err != nil {
		return false
	}
	return json.Unmarshal([]byte(raw), dst) == nil
}

func SaveJSON(db *sql.DB, key string, data any, updatedAt string) error {
	if db == nil {
		return sql.ErrConnDone
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO app_kv(key,value_json,updated_at) VALUES(?,?,?)
		ON CONFLICT(key) DO UPDATE SET value_json=excluded.value_json, updated_at=excluded.updated_at`, key, string(b), updatedAt)
	return err
}
