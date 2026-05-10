(() => {
  const $ = (id) => document.getElementById(id);
  const token = () => localStorage.getItem("flyteam_user_token") || localStorage.getItem("user_token") || "";
  const csrf = () => sessionStorage.getItem("flyteam_user_csrf") || "";
  const escapeHTML = (s) => String(s || "").replace(/[&<>'"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" }[c]));
  const state = { article: null, authed: false };

  async function initCommunityAuthUI() {
    const headers = {};
    if (token()) headers["X-User-Token"] = token();
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

  async function fetchJSON(url, options = {}) {
    const headers = { ...(options.headers || {}) };
    if (token()) headers["X-User-Token"] = token();
    if (csrf()) headers["X-CSRF-Token"] = csrf();
    if (options.body && !headers["Content-Type"]) headers["Content-Type"] = "application/json";
    const res = await fetch(url, { credentials: "same-origin", cache: "no-store", ...options, headers });
    const raw = await res.text();
    let data = null;
    try { data = raw ? JSON.parse(raw) : null; } catch { data = null; }
    if (!res.ok) throw new Error((data && data.detail) || raw || `HTTP ${res.status}`);
    return data || {};
  }
  function createNode(tag, className, text) { const node = document.createElement(tag); if (className) node.className = className; if (text !== undefined) node.textContent = text; return node; }
  function articleIDFromPath() { const parts = window.location.pathname.split("/").filter(Boolean); return parts.length >= 2 && parts[0] === "blog" ? decodeURIComponent(parts[1]) : ""; }
  function formatDate(value) { if (!value) return "未发布"; const date = new Date(value); if (Number.isNaN(date.getTime())) return value; return date.toLocaleString("zh-CN", { year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" }); }
  function initials(text) { return String(text || "F").trim().slice(0, 1).toUpperCase() || "F"; }

  function appendInlineMarkdown(parent, text) {
    const value = String(text || "");
    const regex = /(\[[^\]]+\]\([^\)]+\)|`[^`]+`|\*\*[^*]+\*\*)/g;
    let lastIndex = 0;
    let match;
    while ((match = regex.exec(value)) !== null) {
      if (match.index > lastIndex) parent.appendChild(document.createTextNode(value.slice(lastIndex, match.index)));
      const tokenText = match[0];
      const link = tokenText.match(/^\[([^\]]+)\]\(([^\)]+)\)$/);
      if (link) { const a = document.createElement("a"); a.href = link[2]; a.textContent = link[1]; a.rel = "noopener noreferrer"; parent.appendChild(a); }
      else if (tokenText.startsWith("`")) parent.appendChild(createNode("code", "", tokenText.slice(1, -1)));
      else parent.appendChild(createNode("strong", "", tokenText.slice(2, -2)));
      lastIndex = regex.lastIndex;
    }
    if (lastIndex < value.length) parent.appendChild(document.createTextNode(value.slice(lastIndex)));
  }

  function renderMarkdown(markdown) {
    const root = createNode("div", "article-body campus-reader-body");
    const headings = [];
    const lines = String(markdown || "").split(/\r?\n/);
    let paragraph = [], list = null, inCode = false, codeLang = "", codeLines = [];
    function flushList() { if (list) { root.appendChild(list); list = null; } }
    function flushParagraph() { const text = paragraph.join("\n").trim(); paragraph = []; if (!text) return; flushList(); const p = document.createElement("p"); appendInlineMarkdown(p, text); root.appendChild(p); }
    function flushCode() { flushList(); const pre = document.createElement("pre"); if (codeLang) pre.appendChild(createNode("span", "code-lang", codeLang)); const code = document.createElement("code"); code.textContent = codeLines.join("\n"); pre.appendChild(code); root.appendChild(pre); codeLang = ""; codeLines = []; }
    lines.forEach((line) => {
      const codeFence = line.match(/^```\s*(.*)$/);
      if (codeFence) { if (inCode) { flushCode(); inCode = false; } else { flushParagraph(); flushList(); inCode = true; codeLang = codeFence[1].trim(); } return; }
      if (inCode) { codeLines.push(line); return; }
      const trimmed = line.trim();
      if (!trimmed) { flushParagraph(); flushList(); return; }
      const heading = trimmed.match(/^(#{1,3})\s+(.+)$/);
      if (heading) { flushParagraph(); flushList(); const level = Math.min(3, heading[1].length + 1); const h = document.createElement(`h${level}`); h.textContent = heading[2]; h.id = `section-${headings.length + 1}`; headings.push({ id: h.id, text: heading[2] }); root.appendChild(h); return; }
      const quote = trimmed.match(/^>\s+(.+)$/);
      if (quote) { flushParagraph(); flushList(); const block = document.createElement("blockquote"); appendInlineMarkdown(block, quote[1]); root.appendChild(block); return; }
      const image = trimmed.match(/^!\[(.*?)\]\((.*?)\)$/);
      if (image) { flushParagraph(); flushList(); const fig = document.createElement("figure"); const img = document.createElement("img"); img.alt = image[1] || "文章图片"; img.src = image[2] || ""; fig.appendChild(img); if (image[1]) fig.appendChild(createNode("figcaption", "", image[1])); root.appendChild(fig); return; }
      const li = trimmed.match(/^[-*]\s+(.+)$/);
      if (li) { flushParagraph(); if (!list) list = document.createElement("ul"); const item = document.createElement("li"); appendInlineMarkdown(item, li[1]); list.appendChild(item); return; }
      paragraph.push(line);
    });
    if (inCode) flushCode();
    flushParagraph(); flushList();
    return { node: root, headings };
  }
  function renderToc(headings) { const root = $("articleToc"); if (!root) return; root.innerHTML = headings.length ? headings.map((h) => `<a href="#${escapeHTML(h.id)}">${escapeHTML(h.text)}</a>`).join("") : "<span>这篇文章暂无标题目录</span>"; }

  function renderArticle(article) {
    state.article = article;
    document.title = `${article.title || "文章详情"} - Flyteam`;
    const hero = $("articleHero"), root = $("articleRoot"), author = $("articleAuthorCard");
    const authorName = article.author_nickname || article.author_user_id || "Flyteamer";
    if (hero) {
      hero.style.setProperty("--article-cover", article.cover_url ? `url('${article.cover_url.replace(/'/g, "%27")}')` : "none");
      hero.classList.toggle("has-cover", Boolean(article.cover_url));
      hero.innerHTML = `<p class="campus-pill">${escapeHTML(article.category || article.language || "ARTICLE")}</p><h1>${escapeHTML(article.title || "未命名文章")}</h1><p>${escapeHTML(article.summary || "Flyteam 成员文章")}</p><div class="article-hero-meta"><span>${escapeHTML(authorName)}</span><span>${escapeHTML(formatDate(article.published_at || article.created_at))}</span><span>${Number(article.views || 0)} 阅读</span><span>${Number(article.likes || 0)} 赞</span></div>`;
    }
    if (author) author.innerHTML = `<div class="author-avatar-big">${escapeHTML(initials(authorName))}</div><strong>${escapeHTML(authorName)}</strong><span>@${escapeHTML(article.author_user_id || "flyteam")}</span><p>${Number(article.views || 0)} 阅读 · ${Number(article.comments || 0)} 评论</p>`;
    const authorLink = $("articleAuthorLink");
    if (authorLink && article.author_user_id) authorLink.href = `/space/${encodeURIComponent(article.author_user_id)}`;
    if (!root) return;
    root.innerHTML = "";
    const meta = createNode("div", "reader-meta");
    meta.innerHTML = `<span>${escapeHTML(article.category || article.language || "技术")}</span><span>${escapeHTML(formatDate(article.published_at || article.created_at))}</span><span>${Number(article.favorites || 0)} 收藏</span>`;
    root.appendChild(meta);
    root.appendChild(createNode("h1", "reader-title", article.title || "未命名文章"));
    if (Array.isArray(article.tags) && article.tags.length) { const tags = createNode("div", "reader-tags"); article.tags.forEach((tag) => tags.appendChild(createNode("span", "", tag))); root.appendChild(tags); }
    if (article.summary) root.appendChild(createNode("p", "reader-summary", article.summary));
    const rendered = renderMarkdown(article.content_markdown || "");
    root.appendChild(rendered.node);
    renderToc(rendered.headings);
  }

  async function initArticle() {
    const root = $("articleRoot");
    const id = articleIDFromPath();
    if (!id) { if (root) root.textContent = "缺少文章 ID。"; return; }
    try { const data = await fetchJSON(`/api/blog/articles/${encodeURIComponent(id)}`); renderArticle(data.article || {}); fetchJSON(`/api/blog/articles/${encodeURIComponent(id)}/view`, { method: "POST" }).catch(() => {}); }
    catch (err) { if (root) root.innerHTML = `<div class="campus-empty"><h3>文章加载失败</h3><p>${escapeHTML(err.message || "请稍后刷新重试")}</p></div>`; }
  }

  function setTip(text) { const el = $("articleInteractionTip"); if (el) el.textContent = text || ""; }
  async function loadComments() {
    const articleId = articleIDFromPath();
    const root = $("articleComments");
    if (!root || !articleId) return;
    try {
      const data = await fetchJSON(`/api/blog/articles/${encodeURIComponent(articleId)}/comments`);
      const items = Array.isArray(data.items) ? data.items : [];
      root.innerHTML = items.length ? items.map((c) => { const a = c.author || {}; return `<article class="campus-comment"><div class="comment-avatar">${escapeHTML(initials(a.nickname || a.id))}</div><div><strong>${escapeHTML(a.nickname || a.id || "Flyteamer")}</strong><time>${escapeHTML(formatDate(c.created_at))}</time><p>${escapeHTML(c.content)}</p></div></article>`; }).join("") : '<p class="reader-tip">暂无评论。</p>';
      const counter = $("articleCommentCount");
      if (counter) counter.textContent = String(data.total || items.length);
    } catch (err) { setTip(`评论加载失败：${err.message}`); }
  }
  async function sendComment() {
    const input = $("articleCommentInput");
    const articleId = articleIDFromPath();
    if (!input || !articleId) return;
    const content = input.value.trim();
    if (!content) return;
    try { await fetchJSON(`/api/blog/articles/${encodeURIComponent(articleId)}/comments`, { method: "POST", body: JSON.stringify({ content }) }); input.value = ""; await loadComments(); setTip("评论已发布。"); }
    catch (err) { setTip(`评论失败：${err.message}`); }
  }
  function bindReaction(id, path, activeText, inactiveText) {
    const btn = $(id), articleId = articleIDFromPath();
    if (!btn || !articleId) return;
    let active = false;
    btn.addEventListener("click", async () => {
      try { const data = await fetchJSON(`/api/blog/articles/${encodeURIComponent(articleId)}/${path}`, { method: active ? "DELETE" : "POST" }); active = Boolean(data[path === "like" ? "liked" : "favorited"]); btn.textContent = active ? activeText : inactiveText; setTip(active ? `${activeText}成功。` : `已取消${inactiveText}。`); }
      catch (err) { setTip(`${inactiveText}失败：${err.message}`); }
    });
  }

  $("articleCommentSubmit")?.addEventListener("click", sendComment);
  $("articleCommentInput")?.addEventListener("keydown", (event) => { if (event.key === "Enter" && (event.ctrlKey || event.metaKey)) sendComment(); });
  bindReaction("articleLikeBtn", "like", "已点赞", "点赞");
  bindReaction("articleFavoriteBtn", "favorite", "已收藏", "收藏");
  initCommunityAuthUI();
  initArticle();
  loadComments();
})();
