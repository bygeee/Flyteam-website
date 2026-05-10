package blog

import "strings"

type ArticleRequest struct {
	Title           string   `json:"title"`
	Summary         string   `json:"summary"`
	CoverURL        string   `json:"cover_url"`
	ContentMarkdown string   `json:"content_markdown"`
	Tags            []string `json:"tags"`
	Category        string   `json:"category"`
	Language        string   `json:"language"`
	Status          string   `json:"status"`
}

type Article struct {
	ID              string
	AuthorID        string
	AuthorUserID    string
	AuthorNickname  string
	Title           string
	Summary         string
	CoverURL        string
	ContentMarkdown string
	Status          string
	Visibility      string
	Language        string
	Category        string
	Views           int
	Likes           int
	Favorites       int
	Comments        int
	CreatedAt       string
	UpdatedAt       string
	PublishedAt     string
	Tags            []string
}

func NormalizeStatus(status string) string {
	if strings.EqualFold(strings.TrimSpace(status), "published") {
		return "published"
	}
	return "draft"
}

func NormalizeTags(tags []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, tag := range tags {
		clean := strings.TrimSpace(tag)
		key := strings.ToLower(clean)
		if clean == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, clean)
		if len(out) >= 8 {
			break
		}
	}
	return out
}

func BuildArticleFromRequest(req ArticleRequest) (Article, string) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return Article{}, "Title is required."
	}
	if len([]rune(title)) > 120 {
		return Article{}, "Title must be 120 characters or fewer."
	}
	content := strings.TrimSpace(req.ContentMarkdown)
	if content == "" {
		return Article{}, "Content is required."
	}
	if len([]rune(content)) > 100000 {
		return Article{}, "Content is too long."
	}
	summary := strings.TrimSpace(req.Summary)
	if len([]rune(summary)) > 240 {
		summary = string([]rune(summary)[:240])
	}
	return Article{Title: title, Summary: summary, CoverURL: strings.TrimSpace(req.CoverURL), ContentMarkdown: content, Tags: NormalizeTags(req.Tags), Category: strings.TrimSpace(req.Category), Language: strings.TrimSpace(req.Language), Status: NormalizeStatus(req.Status), Visibility: "public"}, ""
}

func PublicArticle(a Article, includeContent bool) map[string]any {
	out := map[string]any{
		"id":              a.ID,
		"author_id":       a.AuthorID,
		"author_user_id":  a.AuthorUserID,
		"author_nickname": a.AuthorNickname,
		"title":           a.Title,
		"summary":         a.Summary,
		"cover_url":       a.CoverURL,
		"tags":            a.Tags,
		"category":        a.Category,
		"language":        a.Language,
		"status":          a.Status,
		"visibility":      a.Visibility,
		"views":           a.Views,
		"likes":           a.Likes,
		"favorites":       a.Favorites,
		"comments":        a.Comments,
		"created_at":      a.CreatedAt,
		"updated_at":      a.UpdatedAt,
		"published_at":    a.PublishedAt,
	}
	if includeContent {
		out["content_markdown"] = a.ContentMarkdown
	}
	return out
}
