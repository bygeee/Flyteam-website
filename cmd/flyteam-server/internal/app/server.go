package app

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	BaseDir               string
	StorageDir            string
	UploadDir             string
	StaticDir             string
	ImageUploadDir        string
	AwardUploadDir        string
	SeniorUploadDir       string
	ReviewUploadDir       string
	NewsUploadDir         string
	BlogUploadDir         string
	AvatarUploadDir       string
	DatabaseFile          string
	RagIndexFile          string
	TeamContentFile       string
	RecruitContentFile    string
	IngestIndexFile       string
	AdminUsersFile        string
	DefaultDataFiles      []string
	OpenAIAPIKey          string
	OpenAIBaseURL         string
	EmbeddingModel        string
	ChatModel             string
	EmbeddingBatchSize    int
	RetrievalMinRelevance float64
	AdminToken            string
	AdminPassword         string
	AdminSessionHours     int
	UserSessionHours      int
	AdminCookieSecure     bool
	MaxUploadFiles        int
	MaxImageUploadBytes   int64
	MaxPDFUploadBytes     int64
	ListenAddr            string
}

type Server struct {
	cfg       Config
	rag       *RagService
	db        *sql.DB
	sessions  map[string]AdminSession
	sessMu    sync.Mutex
	rate      map[string][]time.Time
	rateMu    sync.Mutex
	captchas  map[string]CaptchaEntry
	captchaMu sync.Mutex
}

func Run() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	for _, dir := range []string{cfg.StorageDir, cfg.UploadDir, cfg.ImageUploadDir, cfg.AwardUploadDir, cfg.SeniorUploadDir, cfg.ReviewUploadDir, cfg.NewsUploadDir, cfg.BlogUploadDir, cfg.AvatarUploadDir, filepath.Dir(cfg.RagIndexFile), filepath.Dir(cfg.DatabaseFile)} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	db, err := openDatabase(cfg)
	if err != nil {
		log.Fatalf("database init failed: %v", err)
	}
	defer db.Close()
	rag := NewRagService(cfg, db)
	s := &Server{
		cfg:      cfg,
		rag:      rag,
		db:       db,
		sessions: map[string]AdminSession{},
		rate:     map[string][]time.Time{},
		captchas: map[string]CaptchaEntry{},
	}
	log.Printf("Flyteam Go server listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, s); err != nil {
		log.Fatal(err)
	}
}

func LoadConfig() (Config, error) {
	base, err := os.Getwd()
	if err != nil {
		return Config{}, err
	}
	loadDotEnv(filepath.Join(base, ".env"))
	atoi := func(key string, def int) int {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			return def
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return def
		}
		return n
	}
	atof := func(key string, def float64) float64 {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			return def
		}
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return def
		}
		return n
	}
	truthy := func(key string) bool {
		v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
		return v == "1" || v == "true" || v == "yes" || v == "on"
	}
	storage := filepath.Join(base, "storage")
	upload := filepath.Join(storage, "uploads")
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	maxImgMB := atoi("MAX_IMAGE_UPLOAD_MB", 8)
	maxPDFMB := atoi("MAX_PDF_UPLOAD_MB", 25)
	return Config{
		BaseDir:               base,
		StorageDir:            storage,
		UploadDir:             upload,
		StaticDir:             filepath.Join(base, "app", "static"),
		ImageUploadDir:        filepath.Join(upload, "images"),
		AwardUploadDir:        filepath.Join(upload, "awards"),
		SeniorUploadDir:       filepath.Join(upload, "seniors"),
		ReviewUploadDir:       filepath.Join(upload, "review"),
		NewsUploadDir:         filepath.Join(upload, "news"),
		BlogUploadDir:         filepath.Join(upload, "blog"),
		AvatarUploadDir:       filepath.Join(upload, "avatars"),
		DatabaseFile:          getenv("DATABASE_FILE", filepath.Join(storage, "flyteam.db")),
		RagIndexFile:          filepath.Join(storage, "rag_index_go.json"),
		TeamContentFile:       filepath.Join(storage, "team_content.json"),
		RecruitContentFile:    filepath.Join(storage, "recruit_applications.json"),
		IngestIndexFile:       filepath.Join(storage, "ingest_index.json"),
		AdminUsersFile:        filepath.Join(storage, "admin_users.json"),
		DefaultDataFiles:      []string{filepath.Join(upload, "flyteam_knowledge.pdf")},
		OpenAIAPIKey:          apiKey,
		OpenAIBaseURL:         getenv("OPENAI_BASE_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1"),
		EmbeddingModel:        getenv("EMBEDDING_MODEL", "text-embedding-v4"),
		ChatModel:             getenv("CHAT_MODEL", "qwen-plus"),
		EmbeddingBatchSize:    atoi("EMBEDDING_BATCH_SIZE", 10),
		RetrievalMinRelevance: atof("RETRIEVAL_MIN_RELEVANCE", 0.08),
		AdminToken:            os.Getenv("ADMIN_TOKEN"),
		AdminPassword:         os.Getenv("ADMIN_PASSWORD"),
		AdminSessionHours:     atoi("ADMIN_SESSION_HOURS", 8),
		UserSessionHours:      atoi("USER_SESSION_HOURS", 168),
		AdminCookieSecure:     truthy("ADMIN_COOKIE_SECURE"),
		MaxUploadFiles:        atoi("MAX_UPLOAD_FILES", 20),
		MaxImageUploadBytes:   int64(max(1, maxImgMB)) * 1024 * 1024,
		MaxPDFUploadBytes:     int64(max(1, maxPDFMB)) * 1024 * 1024,
		ListenAddr:            getenv("LISTEN_ADDR", ":"+getenv("PORT", "8000")),
	}, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func loadDotEnv(path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	text := strings.TrimPrefix(string(b), "\ufeff")
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		val = strings.Trim(val, "\"'")
		if key != "" {
			_ = os.Setenv(key, val)
		}
	}
}
