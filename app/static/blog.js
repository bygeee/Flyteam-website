(() => {
  const $ = (id) => document.getElementById(id);
  const escapeHTML = (s) => String(s || "").replace(/[&<>'"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" }[c]));
  const state = { articles: [], recommendations: [] };

  async function fetchJSON(url, options = {}) {
    const res = await fetch(url, { credentials: "same-origin", ...options });
    const raw = await res.text();
    let data = null;
    try { data = raw ? JSON.parse(raw) : null; } catch { data = null; }
    if (!res.ok) throw new Error((data && data.detail) || raw || `HTTP ${res.status}`);
    return data || {};
  }

  function formatDate(value) {
    if (!value) return "未发布";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return date.toLocaleDateString("zh-CN", { year: "numeric", month: "2-digit", day: "2-digit" });
  }

  function normalizeArticle(item = {}) {
    const author = item.author || {};
    return {
      id: item.id || "",
      title: item.title || "未命名文章",
      summary: item.summary || "作者还没有填写摘要，点进文章看看正文内容。",
      cover: item.cover_url || "",
      category: item.category || item.language || "技术随笔",
      tags: Array.isArray(item.tags) ? item.tags : [],
      views: Number(item.views || 0),
      likes: Number(item.likes || 0),
      favorites: Number(item.favorites || 0),
      comments: Number(item.comments || 0),
      publishedAt: item.published_at || item.created_at || "",
      authorID: item.author_user_id || author.user_id || author.id || "",
      authorName: item.author_nickname || author.nickname || author.id || item.author_user_id || "Flyteamer",
    };
  }

  function initials(name) {
    const text = String(name || "F").trim();
    return (text[0] || "F").toUpperCase();
  }

  function articleHref(a) {
    return `/blog/${encodeURIComponent(a.id)}`;
  }

  function renderFeedArticle(raw, index) {
    const a = normalizeArticle(raw);
    const tags = a.tags.length ? a.tags : [a.category].filter(Boolean);
    const cover = a.cover
      ? `<img class="blog-post-cover" src="${escapeHTML(a.cover)}" alt="${escapeHTML(a.title)}">`
      : `<div class="blog-post-cover fallback"><span>${escapeHTML(initials(a.title))}</span></div>`;
    return `
      <article class="blog-post-card" style="--post-index:${index + 1}">
        <a class="blog-post-media" href="${articleHref(a)}">${cover}</a>
        <div class="blog-post-body">
          <div class="blog-post-meta"><span>${escapeHTML(a.category)}</span><span>${escapeHTML(formatDate(a.publishedAt))}</span><span>${escapeHTML(a.authorName)}</span></div>
          <h3><a href="${articleHref(a)}">${escapeHTML(a.title)}</a></h3>
          <p>${escapeHTML(a.summary)}</p>
          <div class="blog-post-footer">
            <div class="blog-tags-inline">${tags.slice(0, 4).map((tag) => `<span>${escapeHTML(tag)}</span>`).join("")}</div>
            <div class="blog-post-stats"><span>👁 ${a.views}</span><span>👍 ${a.likes}</span><span>💬 ${a.comments}</span></div>
          </div>
        </div>
      </article>`;
  }

  function renderCompactArticle(raw, rank = 0) {
    const a = normalizeArticle(raw);
    return `<a class="blog-rank-item" href="${articleHref(a)}"><b>${rank + 1}</b><span><strong>${escapeHTML(a.title)}</strong><em>${a.views} 阅读 · ${a.likes} 赞</em></span></a>`;
  }

  function updateStats(articles) {
    const total = $("blogArticleTotal");
    const reads = $("blogReadTotal");
    if (total) total.textContent = String(articles.length);
    if (reads) reads.textContent = String(articles.reduce((sum, item) => sum + Number(item.views || 0), 0));
    const tags = new Map();
    articles.forEach((item) => (Array.isArray(item.tags) ? item.tags : []).forEach((tag) => {
      const clean = String(tag || "").trim();
      if (clean) tags.set(clean, (tags.get(clean) || 0) + 1);
    }));
    const tagRoot = $("blogTags");
    if (tagRoot && tags.size) {
      tagRoot.innerHTML = [...tags.entries()].sort((a, b) => b[1] - a[1]).slice(0, 18).map(([tag]) => `<span>${escapeHTML(tag)}</span>`).join("");
    }
  }

  async function initBlogList() {
    const list = $("blogList");
    const status = $("blogStatus");
    if (!list || !status) return;
    list.innerHTML = Array.from({ length: 4 }).map(() => '<div class="blog-skeleton"></div>').join("");
    try {
      const data = await fetchJSON("/api/blog/articles?page_size=50");
      const articles = Array.isArray(data.articles) ? data.articles : (Array.isArray(data.items) ? data.items : []);
      state.articles = articles;
      updateStats(articles);
      if (!articles.length) {
        status.textContent = "暂时还没有公开文章。";
        list.innerHTML = `<div class="blog-empty"><h3>还没有文章</h3><p>登录后发布第一篇 Flyteam 技术博客吧。</p><a class="community-btn primary" href="/editor">去写文章</a></div>`;
        return;
      }
      status.textContent = `共 ${data.total ?? articles.length} 篇公开文章，按发布时间倒序展示`;
      list.innerHTML = articles.map(renderFeedArticle).join("");
    } catch (err) {
      status.textContent = err.message || "文章加载失败";
      list.innerHTML = `<div class="blog-empty"><h3>加载失败</h3><p>${escapeHTML(err.message || "请稍后刷新重试")}</p></div>`;
    }
  }

  async function loadRecommendations() {
    const root = $("blogRecommendations");
    if (!root) return;
    try {
      const data = await fetchJSON("/api/blog/recommendations?limit=8");
      const items = Array.isArray(data.items) ? data.items : [];
      state.recommendations = items;
      root.innerHTML = items.length ? items.map(renderCompactArticle).join("") : '<p class="blog-muted">暂无推荐文章。</p>';
    } catch (err) {
      root.innerHTML = `<p class="blog-muted">推荐加载失败：${escapeHTML(err.message)}</p>`;
    }
  }

  function renderSearchArticle(raw) {
    const a = normalizeArticle(raw);
    return `<a class="blog-search-hit" href="${articleHref(a)}"><strong>${escapeHTML(a.title)}</strong><span>${escapeHTML(a.authorName)} · ${a.views} 阅读</span></a>`;
  }

  async function runSearch() {
    const input = $("communitySearchInput");
    const root = $("communitySearchResults");
    if (!input || !root) return;
    const q = input.value.trim();
    if (!q) { root.innerHTML = ""; return; }
    root.innerHTML = '<p class="blog-muted">搜索中...</p>';
    try {
      const data = await fetchJSON(`/api/search?q=${encodeURIComponent(q)}`);
      const articles = Array.isArray(data.articles) ? data.articles : [];
      const users = Array.isArray(data.users) ? data.users : [];
      root.innerHTML = `
        <div class="blog-search-section"><h4>文章</h4>${articles.map(renderSearchArticle).join("") || '<p class="blog-muted">暂无文章。</p>'}</div>
        <div class="blog-search-section"><h4>用户</h4>${users.map((u) => `<a class="blog-search-hit" href="/space/${encodeURIComponent(u.user_id || u.id || "")}"><strong>${escapeHTML(u.nickname || u.id)}</strong><span>@${escapeHTML(u.user_id || u.id)}</span></a>`).join("") || '<p class="blog-muted">暂无用户。</p>'}</div>`;
    } catch (err) {
      root.innerHTML = `<p class="blog-muted">搜索失败：${escapeHTML(err.message)}</p>`;
    }
  }

  $("communitySearchInput")?.addEventListener("keydown", (event) => { if (event.key === "Enter") runSearch(); });
  $("communitySearchBtn")?.addEventListener("click", runSearch);
  initBlogList();
  loadRecommendations();
})();
