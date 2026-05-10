(() => {
  const $ = (id) => document.getElementById(id);
  const escapeHTML = (s) => String(s || "").replace(/[&<>'"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" }[c]));
  const state = { articles: [], recommendations: [], sort: "latest", filter: "all", authed: false };

  const userToken = () => localStorage.getItem("flyteam_user_token") || localStorage.getItem("user_token") || "";

  async function initCommunityAuthUI() {
    const headers = {};
    if (userToken()) headers["X-User-Token"] = userToken();
    let authed = false;
    try {
      const res = await fetch("/api/users/me", { credentials: "same-origin", cache: "no-store", headers });
      if (res.ok) {
        authed = true;
        const data = await res.json().catch(() => ({}));
        if (data && data.user) localStorage.setItem("flyteam_user", JSON.stringify(data.user));
      }
    } catch {
      authed = false;
    }
    state.authed = authed;
    document.body.classList.toggle("community-logged-in", authed);
    document.body.classList.toggle("community-guest", !authed);
  }

  function writeLinkHTML(label = "去写文章", className = "campus-primary") {
    return state.authed
      ? `<a class="${className}" href="/editor">${escapeHTML(label)}</a>`
      : `<a class="${className}" href="/user-login?next=/editor">登录后写文章</a>`;
  }

  async function fetchJSON(url, options = {}) {
    const res = await fetch(url, { credentials: "same-origin", cache: "no-store", ...options });
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
  function dateScore(value) { const t = new Date(value || 0).getTime(); return Number.isFinite(t) ? t : 0; }

  function normalizeArticle(item = {}) {
    const author = item.author || {};
    return {
      id: item.id || "",
      title: item.title || "未命名文章",
      summary: item.summary || "作者还没有填写摘要，点进文章看看正文内容。",
      cover: item.cover_url || "",
      category: item.category || item.language || "技术随笔",
      tags: Array.isArray(item.tags) ? item.tags.filter(Boolean) : [],
      views: Number(item.views || 0),
      likes: Number(item.likes || 0),
      favorites: Number(item.favorites || 0),
      comments: Number(item.comments || 0),
      publishedAt: item.published_at || item.created_at || "",
      authorID: item.author_user_id || author.user_id || author.id || "",
      authorName: item.author_nickname || author.nickname || author.id || item.author_user_id || "Flyteamer",
    };
  }

  function privateHref(path) { return state.authed ? path : `/user-login?next=${encodeURIComponent(path)}`; }
  function articleHref(a) { return privateHref(`/blog/${encodeURIComponent(a.id)}`); }
  function spaceHref(userID) { return privateHref(`/space/${encodeURIComponent(userID || "")}`); }
  function initials(text) { return String(text || "F").trim().slice(0, 1).toUpperCase() || "F"; }
  function heat(a) { return a.views + a.likes * 5 + a.favorites * 8 + a.comments * 3; }
  function minutes(a) { return Math.max(2, Math.ceil(String(a.summary || "").length / 80) + 3); }
  function fallbackCover(a, extra = "") { return `<div class="campus-cover-fallback ${extra}"><span>${escapeHTML(initials(a.title))}</span><i>${escapeHTML(a.category)}</i></div>`; }
  function coverHTML(a, cls = "") { return a.cover ? `<img class="${cls}" src="${escapeHTML(a.cover)}" alt="${escapeHTML(a.title)}" loading="lazy">` : fallbackCover(a, cls); }
  function tagsHTML(a, max = 4) { return (a.tags.length ? a.tags : [a.category]).slice(0, max).map((tag) => `<span>${escapeHTML(tag)}</span>`).join(""); }

  function updateStats(articles) {
    const normalized = articles.map(normalizeArticle);
    const authors = new Set(normalized.map((a) => a.authorID || a.authorName).filter(Boolean));
    const totals = {
      blogArticleTotal: normalized.length,
      blogReadTotal: normalized.reduce((sum, a) => sum + a.views, 0),
      blogCommentTotal: normalized.reduce((sum, a) => sum + a.comments, 0),
      blogAuthorTotal: authors.size,
    };
    Object.entries(totals).forEach(([id, value]) => { const el = $(id); if (el) el.textContent = String(value); });
    const tagMap = new Map();
    normalized.forEach((a) => [...a.tags, a.category].forEach((tag) => { const clean = String(tag || "").trim(); if (clean) tagMap.set(clean, (tagMap.get(clean) || 0) + 1); }));
    const tagRoot = $("blogTags");
    if (tagRoot && tagMap.size) tagRoot.innerHTML = [...tagMap.entries()].sort((a, b) => b[1] - a[1]).slice(0, 18).map(([tag, count]) => `<span>${escapeHTML(tag)}<em>${count}</em></span>`).join("");
  }

  function filteredArticles() {
    const filter = state.filter.toLowerCase();
    let items = state.articles.map(normalizeArticle);
    if (filter !== "all") items = items.filter((a) => [a.category, a.title, a.summary, ...a.tags].some((v) => String(v || "").toLowerCase().includes(filter)));
    if (state.sort === "views") items.sort((a, b) => b.views - a.views || dateScore(b.publishedAt) - dateScore(a.publishedAt));
    else if (state.sort === "hot") items.sort((a, b) => heat(b) - heat(a) || dateScore(b.publishedAt) - dateScore(a.publishedAt));
    else items.sort((a, b) => dateScore(b.publishedAt) - dateScore(a.publishedAt));
    return items;
  }

  function renderFeatured(a) {
    const root = $("featuredArticle");
    if (!root) return;
    if (!a) { root.innerHTML = ""; return; }
    root.innerHTML = `<article class="campus-feature-card">
      <a class="feature-cover" href="${articleHref(a)}">${coverHTML(a)}</a>
      <div class="feature-copy">
        <div class="campus-card-meta"><span>${escapeHTML(a.category)}</span><span>${escapeHTML(formatDate(a.publishedAt))}</span><span>${minutes(a)} min read</span></div>
        <h3><a href="${articleHref(a)}">${escapeHTML(a.title)}</a></h3>
        <p>${escapeHTML(a.summary)}</p>
        <div class="campus-author-row"><span>${escapeHTML(initials(a.authorName))}</span><b>${escapeHTML(a.authorName)}</b><em>@${escapeHTML(a.authorID || "flyteam")}</em></div>
      </div>
      <div class="feature-stats"><span>👁 ${a.views}</span><span>👍 ${a.likes}</span><span>💬 ${a.comments}</span></div>
    </article>`;
  }

  function renderArticleCard(a, index) {
    return `<article class="campus-article-card ${index % 5 === 0 ? "wide" : ""}" style="--i:${index}">
      <a class="campus-card-cover" href="${articleHref(a)}">${coverHTML(a)}</a>
      <div class="campus-card-body">
        <div class="campus-card-meta"><span>${escapeHTML(a.category)}</span><span>${escapeHTML(formatDate(a.publishedAt))}</span></div>
        <h3><a href="${articleHref(a)}">${escapeHTML(a.title)}</a></h3>
        <p>${escapeHTML(a.summary)}</p>
        <div class="campus-card-tags">${tagsHTML(a)}</div>
        <div class="campus-card-foot"><span>${escapeHTML(a.authorName)}</span><b>${a.views} 阅读</b></div>
      </div>
    </article>`;
  }

  function renderFeed() {
    const list = $("blogList");
    const status = $("blogStatus");
    const heroTitle = $("heroFeatureTitle");
    const heroSummary = $("heroFeatureSummary");
    if (!list) return;
    const items = filteredArticles();
    const [featured, ...rest] = items;
    renderFeatured(featured);
    if (heroTitle && featured) heroTitle.textContent = featured.title;
    if (heroSummary && featured) heroSummary.textContent = featured.summary;
    if (!items.length) {
      if (status) status.textContent = "当前筛选下没有文章。";
      list.innerHTML = `<div class="campus-empty"><h3>还没有文章</h3><p>换一个筛选条件，或者发布第一篇 Flyteam 技术博客。</p>${writeLinkHTML()}</div>`;
      return;
    }
    if (status) status.textContent = `共 ${items.length} 篇文章 · ${state.sort === "hot" ? "按综合热度" : state.sort === "views" ? "按阅读量" : "按发布时间"}展示`;
    list.innerHTML = rest.map(renderArticleCard).join("") || `<div class="campus-empty"><h3>只有一篇文章</h3><p>继续沉淀更多内容，让广场更丰富。</p></div>`;
  }

  function renderRankItem(raw, rank = 0) {
    const a = normalizeArticle(raw);
    return `<a class="campus-rank-item" href="${articleHref(a)}"><b>${String(rank + 1).padStart(2, "0")}</b><span><strong>${escapeHTML(a.title)}</strong><em>${a.views} 阅读 · ${a.likes} 赞 · ${a.comments} 评</em></span></a>`;
  }

  async function initBlogList() {
    const list = $("blogList");
    const status = $("blogStatus");
    if (list) list.innerHTML = Array.from({ length: 6 }).map(() => '<div class="campus-skeleton"></div>').join("");
    try {
      const data = await fetchJSON("/api/blog/articles?page_size=80");
      const articles = Array.isArray(data.articles) ? data.articles : (Array.isArray(data.items) ? data.items : []);
      state.articles = articles;
      updateStats(articles);
      renderFeed();
    } catch (err) {
      if (status) status.textContent = err.message || "文章加载失败";
      if (list) list.innerHTML = `<div class="campus-empty"><h3>加载失败</h3><p>${escapeHTML(err.message || "请稍后刷新重试")}</p></div>`;
    }
  }

  async function loadRecommendations() {
    const root = $("blogRecommendations");
    if (!root) return;
    root.innerHTML = '<div class="campus-mini-loading">推荐加载中...</div>';
    try {
      const data = await fetchJSON("/api/blog/recommendations?limit=8");
      const items = Array.isArray(data.items) ? data.items : [];
      state.recommendations = items;
      root.innerHTML = items.length ? items.map(renderRankItem).join("") : '<p class="campus-muted">暂无推荐文章。</p>';
    } catch (err) { root.innerHTML = `<p class="campus-muted">推荐加载失败：${escapeHTML(err.message)}</p>`; }
  }

  function renderSearchArticle(raw) { const a = normalizeArticle(raw); return `<a class="campus-search-hit" href="${articleHref(a)}"><strong>${escapeHTML(a.title)}</strong><span>${escapeHTML(a.authorName)} · ${a.views} 阅读</span></a>`; }
  async function runSearch() {
    const input = $("communitySearchInput");
    const root = $("communitySearchResults");
    if (!input || !root) return;
    const q = input.value.trim();
    if (!q) { root.innerHTML = ""; return; }
    root.innerHTML = '<p class="campus-muted">搜索中...</p>';
    try {
      const data = await fetchJSON(`/api/search?q=${encodeURIComponent(q)}`);
      const articles = Array.isArray(data.articles) ? data.articles : [];
      const users = Array.isArray(data.users) ? data.users : [];
      const userSection = state.authed
        ? `<div class="campus-search-section"><h4>用户</h4>${users.map((u) => `<a class="campus-search-hit" href="${spaceHref(u.user_id || u.id || "")}"><strong>${escapeHTML(u.nickname || u.id)}</strong><span>@${escapeHTML(u.user_id || u.id)}</span></a>`).join("") || '<p class="campus-muted">暂无用户。</p>'}</div>`
        : '<div class="campus-search-section"><h4>用户</h4><p class="campus-muted">登录后可查看用户主页、关注和私信。</p></div>';
      root.innerHTML = `<div class="campus-search-section"><h4>文章</h4>${articles.map(renderSearchArticle).join("") || '<p class="campus-muted">暂无文章。</p>'}</div>${userSection}`;
    } catch (err) { root.innerHTML = `<p class="campus-muted">搜索失败：${escapeHTML(err.message)}</p>`; }
  }

  document.querySelectorAll("[data-sort]").forEach((btn) => btn.addEventListener("click", () => { state.sort = btn.dataset.sort || "latest"; document.querySelectorAll("[data-sort]").forEach((x) => x.classList.toggle("on", x === btn)); renderFeed(); }));
  document.querySelectorAll("[data-filter]").forEach((btn) => btn.addEventListener("click", () => { state.filter = btn.dataset.filter || "all"; document.querySelectorAll("[data-filter]").forEach((x) => x.classList.toggle("on", x === btn)); renderFeed(); }));
  $("communitySearchInput")?.addEventListener("keydown", (event) => { if (event.key === "Enter") runSearch(); });
  $("communitySearchBtn")?.addEventListener("click", runSearch);
  initCommunityAuthUI().finally(() => {
    initBlogList();
    loadRecommendations();
  });
})();
