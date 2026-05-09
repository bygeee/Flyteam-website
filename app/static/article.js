(() => {
  const token = () => localStorage.getItem("flyteam_user_token") || localStorage.getItem("user_token") || "";
  const headers = () => token() ? { "X-User-Token": token(), "Content-Type": "application/json" } : { "Content-Type": "application/json" };
  const qs = new URLSearchParams(location.search);
  const articleId = document.body?.dataset.articleId || qs.get("id") || location.pathname.split("/").filter(Boolean).pop();
  const $ = (id) => document.getElementById(id);
  async function api(path, options = {}) {
    const res = await fetch(path, { ...options, headers: { ...headers(), ...(options.headers || {}) } });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(data.detail || `HTTP ${res.status}`);
    return data;
  }
  function escapeHTML(s) { return String(s || "").replace(/[&<>'"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" }[c])); }
  function setTip(text) { const el = $("articleInteractionTip"); if (el) el.textContent = text || ""; }
  async function loadComments() {
    const root = $("articleComments");
    if (!root || !articleId) return;
    try {
      const data = await api(`/api/blog/articles/${encodeURIComponent(articleId)}/comments`);
      const items = Array.isArray(data.items) ? data.items : [];
      root.innerHTML = items.length ? items.map((c) => `<article class="comment-item" data-id="${escapeHTML(c.id)}"><b>${escapeHTML((c.author || {}).nickname || (c.author || {}).id)}</b><p>${escapeHTML(c.content)}</p></article>`).join("") : '<p class="section-sub">暂无评论。</p>';
      const counter = $("articleCommentCount");
      if (counter) counter.textContent = String(data.total || items.length);
    } catch (err) { setTip(`评论加载失败：${err.message}`); }
  }
  async function sendComment() {
    const input = $("articleCommentInput");
    if (!input || !articleId) return;
    const content = input.value.trim();
    if (!content) return;
    try {
      await api(`/api/blog/articles/${encodeURIComponent(articleId)}/comments`, { method: "POST", body: JSON.stringify({ content }) });
      input.value = "";
      await loadComments();
    } catch (err) { setTip(`评论失败：${err.message}`); }
  }
  function bindReaction(id, path, activeText, inactiveText) {
    const btn = $(id);
    if (!btn || !articleId) return;
    let active = false;
    btn.addEventListener("click", async () => {
      try {
        const method = active ? "DELETE" : "POST";
        const data = await api(`/api/blog/articles/${encodeURIComponent(articleId)}/${path}`, { method });
        active = Boolean(data[path === "like" ? "liked" : "favorited"]);
        btn.textContent = active ? activeText : inactiveText;
      } catch (err) { setTip(`${inactiveText}失败：${err.message}`); }
    });
  }
  $("articleCommentSubmit")?.addEventListener("click", sendComment);
  $("articleCommentInput")?.addEventListener("keydown", (event) => { if (event.key === "Enter" && (event.ctrlKey || event.metaKey)) sendComment(); });
  bindReaction("articleLikeBtn", "like", "已点赞", "点赞");
  bindReaction("articleFavoriteBtn", "favorite", "已收藏", "收藏");
  loadComments();
})();
