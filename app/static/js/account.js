(() => {
  const $ = (id) => document.getElementById(id);
  const state = { user: null, stats: {}, avatarVersion: Date.now() };
  const token = () => localStorage.getItem("flyteam_user_token") || localStorage.getItem("user_token") || "";
  const csrf = () => sessionStorage.getItem("flyteam_user_csrf") || "";
  const escapeHTML = (s) => String(s || "").replace(/[&<>'"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" }[c]));
  function installStrayCaretGuard() {
    const editableSelector = "input, textarea, select, [contenteditable='true'], [contenteditable='plaintext-only'], .article-body, pre, code";
    const isEditableTarget = (target) => !!(target && target.closest && target.closest(editableSelector));
    const clearCollapsedSelection = (target) => {
      if (isEditableTarget(target)) return;
      const sel = window.getSelection && window.getSelection();
      if (sel && sel.rangeCount && sel.isCollapsed) sel.removeAllRanges();
    };
    document.addEventListener("pointerup", (event) => {
      window.requestAnimationFrame(() => clearCollapsedSelection(event.target));
    }, true);
    document.addEventListener("selectionchange", () => {
      const active = document.activeElement;
      if (isEditableTarget(active)) return;
      const sel = window.getSelection && window.getSelection();
      if (sel && sel.rangeCount && sel.isCollapsed) sel.removeAllRanges();
    });
  }
  function setText(id, text) { const el = $(id); if (el) el.textContent = text || ""; }
  function firstLetter(u) { return String((u && (u.nickname || u.user_id || u.id)) || "F").slice(0, 1).toUpperCase(); }
  function userID(u) { return (u && (u.user_id || u.id)) || ""; }
  async function api(path, options = {}) {
    const headers = { ...(options.headers || {}) };
    if (token()) headers["X-User-Token"] = token();
    if (csrf()) headers["X-CSRF-Token"] = csrf();
    if (options.body && !(options.body instanceof FormData) && !headers["Content-Type"]) headers["Content-Type"] = "application/json";
    const res = await fetch(path, { credentials: "same-origin", cache: "no-store", ...options, headers });
    const raw = await res.text();
    let data = null;
    try { data = raw ? JSON.parse(raw) : null; } catch { data = null; }
    if (!res.ok) throw new Error((data && data.detail) || raw || `HTTP ${res.status}`);
    return data || {};
  }
  function avatarDisplayURL(url) {
    const clean = String(url || "").trim();
    if (!clean) return "";
    const joiner = clean.includes("?") ? "&" : "?";
    return `${clean}${joiner}v=${encodeURIComponent(String(state.avatarVersion || Date.now()))}`;
  }
  function avatarHTML(u, big = false) {
    const src = avatarDisplayURL(u && u.avatar_url);
    if (src) return `<img src="${escapeHTML(src)}" alt="${escapeHTML((u && (u.nickname || userID(u))) || "avatar")}">`;
    return `<span>${escapeHTML(firstLetter(u))}</span>`;
  }
  function renderUser(data) {
    state.user = data.user || data;
    state.stats = data.stats || state.stats || {};
    const u = state.user;
    const uid = userID(u);
    setText("accountDisplayName", u.nickname || uid || "个人中心");
    setText("accountDisplayMeta", uid ? `@${uid} · ${u.bio || "还没有填写简介"}` : "登录后管理资料、账号、安全和内容。")
    setText("statArticles", String(state.stats.articles || 0));
    setText("statFollowers", String(state.stats.followers || 0));
    setText("statFollowing", String(state.stats.following || 0));
    const space = $("accountSpaceLink"); if (space && uid) space.href = `/space/${encodeURIComponent(uid)}`;
    for (const id of ["accountAvatar", "avatarPreview"]) { const el = $(id); if (el) el.innerHTML = avatarHTML(u, id === "avatarPreview"); }
    $("nickname").value = u.nickname || "";
    $("userId").value = uid || "";
    $("avatarUrl").value = u.avatar_url || "";
    $("bio").value = u.bio || "";
    localStorage.setItem("flyteam_user", JSON.stringify(u));
  }
  async function loadMe() {
    if (!token()) { window.location.href = "/user-login?next=/account"; return; }
    try {
      const me = await api("/api/users/me");
      sessionStorage.setItem("flyteam_user_csrf", me.csrf_token || (me.user && me.user.csrf_token) || csrf());
      const profile = await api(`/api/users/${encodeURIComponent(userID(me.user))}`);
      renderUser({ user: me.user, stats: profile.stats || {} });
      loadMyArticles();
    } catch (err) {
      setText("profileMsg", `加载失败：${err.message}`);
    }
  }
  async function saveProfile(event) {
    event.preventDefault();
    setText("profileMsg", "保存中...");
    try {
      const data = await api("/api/users/me/settings", { method: "PUT", body: JSON.stringify({ nickname: $("nickname").value.trim(), user_id: $("userId").value.trim(), avatar_url: $("avatarUrl").value.trim(), bio: $("bio").value.trim() }) });
      renderUser(data);
      setText("profileMsg", "已保存。账号 ID 修改后，主页链接会自动更新。");
    } catch (err) { setText("profileMsg", `保存失败：${err.message}`); }
  }
  async function changePassword(event) {
    event.preventDefault();
    setText("passwordMsg", "修改中...");
    try {
      await api("/api/users/me/password", { method: "PUT", body: JSON.stringify({ old_password: $("oldPassword").value, new_password: $("newPassword").value }) });
      $("oldPassword").value = ""; $("newPassword").value = "";
      setText("passwordMsg", "密码已修改，请牢记新密码。");
    } catch (err) { setText("passwordMsg", `修改失败：${err.message}`); }
  }
  async function uploadAvatar() {
    const file = $("avatarFile").files && $("avatarFile").files[0];
    if (!file) return;
    setText("profileMsg", "头像上传中...");
    const form = new FormData(); form.append("files", file);
    try {
      const data = await api("/api/upload/avatar", { method: "POST", body: form });
      const nextAvatarURL = data.avatar_url || (data.user && data.user.avatar_url) || "";
      const u = { ...(state.user || {}), ...(data.user || {}) };
      if (nextAvatarURL) u.avatar_url = nextAvatarURL;
      state.avatarVersion = Date.now();
      if ($("avatarUrl")) $("avatarUrl").value = u.avatar_url || "";
      renderUser({ user: u, stats: state.stats });
      setText("profileMsg", "头像已立即保存并覆盖旧头像。若浏览器仍显示旧图，请强制刷新页面缓存。");
    } catch (err) { setText("profileMsg", `头像上传失败：${err.message}`); }
    finally { $("avatarFile").value = ""; }
  }
  async function loadMyArticles() {
    const root = $("myArticles"); if (!root) return;
    root.innerHTML = '<p class="account-muted">正在加载文章...</p>';
    try {
      const data = await api("/api/blog/articles?mine=1&page_size=50");
      const items = Array.isArray(data.items) ? data.items : (Array.isArray(data.articles) ? data.articles : []);
      if (!items.length) { root.innerHTML = '<div class="account-empty"><h3>还没有文章</h3><p>去创作中心发布第一篇内容。</p></div>'; return; }
      root.innerHTML = items.map((a) => `<article class="my-article"><div><strong>${escapeHTML(a.title)}</strong><p>${escapeHTML(a.summary || "暂无摘要")}</p><small>${escapeHTML(a.status)} · 浏览 ${Number(a.views || 0)} · 评论 ${Number(a.comments || 0)}</small></div><div><a class="community-btn" href="/blog/${encodeURIComponent(a.id)}">查看</a><a class="community-btn primary" href="/editor?draft=${encodeURIComponent(a.id)}">编辑</a></div></article>`).join("");
    } catch (err) { root.innerHTML = `<p class="account-muted">文章加载失败：${escapeHTML(err.message)}</p>`; }
  }
  function bindTabs() {
    document.querySelectorAll(".account-tabs [data-tab]").forEach((btn) => btn.addEventListener("click", () => {
      document.querySelectorAll(".account-tabs button").forEach((b) => b.classList.toggle("active", b === btn));
      document.querySelectorAll(".account-panel").forEach((panel) => panel.classList.toggle("active", panel.id === `tab-${btn.dataset.tab}`));
    }));
  }
  bindTabs();
  $("profileForm")?.addEventListener("submit", saveProfile);
  $("passwordForm")?.addEventListener("submit", changePassword);
  $("avatarFile")?.addEventListener("change", uploadAvatar);
  $("refreshMyArticles")?.addEventListener("click", loadMyArticles);
  installStrayCaretGuard();
  loadMe();
})();
