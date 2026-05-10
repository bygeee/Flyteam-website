package app

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type blogArticleMeta struct {
	ID         string
	AuthorID   string
	Title      string
	Status     string
	Visibility string
}

type commentRequest struct {
	Content  string `json:"content"`
	ParentID string `json:"parent_id"`
}

type conversationCreateRequest struct {
	TargetUserID string `json:"target_user_id"`
	UserID       string `json:"user_id"`
	RecipientID  string `json:"recipient_id"`
}

type messageCreateRequest struct {
	Content string `json:"content"`
}

type groupCreateRequest struct {
	Name          string   `json:"name"`
	Intro         string   `json:"intro"`
	AvatarURL     string   `json:"avatar_url"`
	Visibility    string   `json:"visibility"`
	MemberUserIDs []string `json:"member_user_ids"`
}

type groupMemberRequest struct {
	UserID string `json:"user_id"`
}

func (s *Server) routeDLCommunityAPI(w http.ResponseWriter, r *http.Request, path string) bool {
	switch {
	case path == "/api/friends" && r.Method == http.MethodGet:
		s.handleListFriends(w, r)
	case strings.HasPrefix(path, "/api/friends/") && r.Method == http.MethodDelete:
		s.handleRemoveFriend(w, r, pathValue(path, "/api/friends/"))
	case path == "/api/friends/requests":
		if r.Method == http.MethodGet {
			s.handleFriendRequests(w, r)
		} else if r.Method == http.MethodPost {
			s.handleCreateFriendRequest(w, r)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		}
	case strings.HasPrefix(path, "/api/friends/requests/") && strings.HasSuffix(path, "/accept") && r.Method == http.MethodPost:
		s.handleFriendRequestAction(w, r, strings.TrimSuffix(pathValue(path, "/api/friends/requests/"), "/accept"), "accept")
	case strings.HasPrefix(path, "/api/friends/requests/") && strings.HasSuffix(path, "/reject") && r.Method == http.MethodPost:
		s.handleFriendRequestAction(w, r, strings.TrimSuffix(pathValue(path, "/api/friends/requests/"), "/reject"), "reject")
	case path == "/api/blog/recommendations" && r.Method == http.MethodGet:
		s.handleBlogRecommendations(w, r)
	case path == "/api/search" && r.Method == http.MethodGet:
		s.handleCommunitySearch(w, r)
	case strings.HasPrefix(path, "/api/blog/articles/") && strings.HasSuffix(path, "/comments"):
		articleID := strings.TrimSuffix(pathValue(path, "/api/blog/articles/"), "/comments")
		if r.Method == http.MethodGet {
			s.handleArticleComments(w, r, articleID)
		} else if r.Method == http.MethodPost {
			s.handleAddArticleComment(w, r, articleID)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		}
	case strings.HasPrefix(path, "/api/blog/comments/"):
		commentID := pathValue(path, "/api/blog/comments/")
		if r.Method == http.MethodPut {
			s.handleUpdateBlogComment(w, r, commentID)
		} else if r.Method == http.MethodDelete {
			s.handleDeleteBlogComment(w, r, commentID)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		}
	case strings.HasPrefix(path, "/api/blog/articles/") && strings.HasSuffix(path, "/like"):
		articleID := strings.TrimSuffix(pathValue(path, "/api/blog/articles/"), "/like")
		if r.Method == http.MethodPost {
			s.handleSetArticleReaction(w, r, articleID, "like", true)
		} else if r.Method == http.MethodDelete {
			s.handleSetArticleReaction(w, r, articleID, "like", false)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		}
	case strings.HasPrefix(path, "/api/blog/articles/") && strings.HasSuffix(path, "/favorite"):
		articleID := strings.TrimSuffix(pathValue(path, "/api/blog/articles/"), "/favorite")
		if r.Method == http.MethodPost {
			s.handleSetArticleReaction(w, r, articleID, "favorite", true)
		} else if r.Method == http.MethodDelete {
			s.handleSetArticleReaction(w, r, articleID, "favorite", false)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		}
	case strings.HasPrefix(path, "/api/social/follows/"):
		target := pathValue(path, "/api/social/follows/")
		if r.Method == http.MethodPost {
			s.handleFollowUser(w, r, target, true)
		} else if r.Method == http.MethodDelete {
			s.handleFollowUser(w, r, target, false)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		}
	case strings.HasPrefix(path, "/api/social/following/") && r.Method == http.MethodGet:
		s.handleFollowList(w, r, pathValue(path, "/api/social/following/"), "following")
	case strings.HasPrefix(path, "/api/social/followers/") && r.Method == http.MethodGet:
		s.handleFollowList(w, r, pathValue(path, "/api/social/followers/"), "followers")
	case path == "/api/messages/conversations":
		if r.Method == http.MethodGet {
			s.handleMessageConversations(w, r)
		} else if r.Method == http.MethodPost {
			s.handleCreateMessageConversation(w, r)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		}
	case strings.HasPrefix(path, "/api/messages/conversations/") && strings.HasSuffix(path, "/messages"):
		conversationID := strings.TrimSuffix(pathValue(path, "/api/messages/conversations/"), "/messages")
		if r.Method == http.MethodGet {
			s.handleConversationMessages(w, r, conversationID)
		} else if r.Method == http.MethodPost {
			s.handleSendConversationMessage(w, r, conversationID)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		}
	case strings.HasPrefix(path, "/api/messages/conversations/") && r.Method == http.MethodGet:
		s.handleMessageConversation(w, r, pathValue(path, "/api/messages/conversations/"))
	case path == "/api/groups":
		if r.Method == http.MethodGet {
			s.handleGroups(w, r)
		} else if r.Method == http.MethodPost {
			s.handleCreateGroup(w, r)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		}
	case strings.HasPrefix(path, "/api/groups/") && strings.Contains(pathValue(path, "/api/groups/"), "/members/") && r.Method == http.MethodDelete:
		rest := pathValue(path, "/api/groups/")
		parts := strings.SplitN(rest, "/members/", 2)
		s.handleRemoveGroupMember(w, r, parts[0], parts[1])
	case strings.HasPrefix(path, "/api/groups/") && strings.HasSuffix(path, "/members"):
		groupID := strings.TrimSuffix(pathValue(path, "/api/groups/"), "/members")
		if r.Method == http.MethodGet {
			s.handleGroupMembers(w, r, groupID)
		} else if r.Method == http.MethodPost {
			s.handleJoinGroup(w, r, groupID)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		}
	case strings.HasPrefix(path, "/api/groups/") && strings.HasSuffix(path, "/messages"):
		groupID := strings.TrimSuffix(pathValue(path, "/api/groups/"), "/messages")
		if r.Method == http.MethodGet {
			s.handleGroupMessages(w, r, groupID)
		} else if r.Method == http.MethodPost {
			s.handleSendGroupMessage(w, r, groupID)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		}
	case strings.HasPrefix(path, "/api/groups/"):
		groupID := pathValue(path, "/api/groups/")
		if r.Method == http.MethodGet {
			s.handleGroup(w, r, groupID)
		} else if r.Method == http.MethodPut {
			s.handleUpdateGroup(w, r, groupID)
		} else if r.Method == http.MethodDelete {
			s.handleDeleteGroup(w, r, groupID)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		}
	case path == "/api/notifications" && r.Method == http.MethodGet:
		s.handleNotifications(w, r)
	case strings.HasPrefix(path, "/api/notifications/") && strings.HasSuffix(path, "/read") && r.Method == http.MethodPost:
		s.handleReadNotification(w, r, strings.TrimSuffix(pathValue(path, "/api/notifications/"), "/read"))
	default:
		return false
	}
	return true
}

func (s *Server) loadArticleMeta(articleID string) (blogArticleMeta, error) {
	articleID = strings.TrimSpace(articleID)
	if articleID == "" || s.db == nil {
		return blogArticleMeta{}, sql.ErrNoRows
	}
	var a blogArticleMeta
	err := s.db.QueryRow(`SELECT id, author_id, title, status, visibility FROM blog_articles WHERE id=?`, articleID).Scan(&a.ID, &a.AuthorID, &a.Title, &a.Status, &a.Visibility)
	return a, err
}

func articleReadable(a blogArticleMeta) bool {
	return a.Status == "published" && a.Visibility == "public"
}

func articleInteractive(a blogArticleMeta) bool {
	return a.Status == "published" && (a.Visibility == "public" || a.Visibility == "followers")
}

func cleanCommunityText(raw string, maxRunes int) (string, error) {
	text := strings.TrimSpace(strings.ReplaceAll(raw, "\x00", ""))
	if text == "" {
		return "", errors.New("content is required")
	}
	if len([]rune(text)) > maxRunes {
		return "", fmt.Errorf("content is too long, max %d characters", maxRunes)
	}
	return text, nil
}

func parsePage(r *http.Request, defaultSize, maxSize int) (page, pageSize, offset int) {
	page = parsePositiveInt(r.URL.Query().Get("page"), 1)
	pageSize = parsePositiveInt(r.URL.Query().Get("page_size"), defaultSize)
	if pageSize > maxSize {
		pageSize = maxSize
	}
	offset = (page - 1) * pageSize
	return
}

func parseLimit(r *http.Request, def, max int) int {
	limit := parsePositiveInt(r.URL.Query().Get("limit"), def)
	if limit > max {
		limit = max
	}
	return limit
}

func parsePositiveInt(raw string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 1 {
		return def
	}
	return n
}

func nullIfEmpty(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return strings.TrimSpace(v)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func previewText(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return string(r)
	}
	return string(r[:max]) + "..."
}
