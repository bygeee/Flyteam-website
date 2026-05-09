(() => {
  const searchInput = document.getElementById("communitySearchInput");
  const searchRoot = document.getElementById("communitySearchResults");
  const recommendRoot = document.getElementById("blogRecommendations");
  function escapeHTML(s) { return String(s || "").replace(/[&<>'"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" }[c])); }
  async function fetchJSON(path) {
    const res = await fetch(path);
    const data = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(data.detail || `HTTP ${res.status}`);
    return data;
  }
  function articleCard(item) {
    const author = item.author || {};
    return `<article class="community-article-card"><h3>${escapeHTML(item.title)}</h3><p>${escapeHTML(item.summary)}</p><small>${escapeHTML(author.nickname || author.id || "匿名")} · 浏览 ${Number(item.views || 0)} · 点赞 ${Number(item.likes || 0)} · 评论 ${Number(item.comments || 0)}</small></article>`;
  }
  async function loadRecommendations() {
    if (!recommendRoot) return;
    try {
      const data = await fetchJSON("/api/blog/recommendations?limit=8");
      const items = Array.isArray(data.items) ? data.items : [];
      recommendRoot.innerHTML = items.length ? items.map(articleCard).join("") : '<p class="section-sub">暂无推荐文章。</p>';
    } catch (err) { recommendRoot.innerHTML = `<p class="section-sub">推荐加载失败：${escapeHTML(err.message)}</p>`; }
  }
  async function runSearch() {
    if (!searchInput || !searchRoot) return;
    const q = searchInput.value.trim();
    if (!q) { searchRoot.innerHTML = ""; return; }
    try {
      const data = await fetchJSON(`/api/search?q=${encodeURIComponent(q)}`);
      const articles = Array.isArray(data.articles) ? data.articles : [];
      const users = Array.isArray(data.users) ? data.users : [];
      searchRoot.innerHTML = `<h3>文章</h3>${articles.map(articleCard).join("") || '<p class="section-sub">暂无文章。</p>'}<h3>用户</h3>${users.map((u) => `<p>${escapeHTML(u.nickname)}（${escapeHTML(u.id)}）</p>`).join("") || '<p class="section-sub">暂无用户。</p>'}`;
    } catch (err) { searchRoot.innerHTML = `<p class="section-sub">搜索失败：${escapeHTML(err.message)}</p>`; }
  }
  searchInput?.addEventListener("keydown", (event) => { if (event.key === "Enter") runSearch(); });
  document.getElementById("communitySearchBtn")?.addEventListener("click", runSearch);
  loadRecommendations();
})();
