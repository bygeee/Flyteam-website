package app

import domain "flyteam-website/cmd/flyteam-server/internal/blog"

type BlogArticleRequest = domain.ArticleRequest
type BlogArticle = domain.Article

func normalizeBlogStatus(status string) string {
	return domain.NormalizeStatus(status)
}

func normalizeBlogTags(tags []string) []string {
	return domain.NormalizeTags(tags)
}

func buildBlogArticleFromRequest(req BlogArticleRequest) (BlogArticle, string) {
	return domain.BuildArticleFromRequest(req)
}

func publicBlogArticle(a BlogArticle, includeContent bool) map[string]any {
	return domain.PublicArticle(a, includeContent)
}
