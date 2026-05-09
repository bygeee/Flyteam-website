(() => {
  const token = () => localStorage.getItem("flyteam_user_token") || localStorage.getItem("user_token") || "";
  async function api(path, options = {}) {
    const headers = { ...(options.headers || {}) };
    if (token()) headers["X-User-Token"] = token();
    if (options.body && !headers["Content-Type"]) headers["Content-Type"] = "application/json";
    const res = await fetch(path, { ...options, headers });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(data.detail || `HTTP ${res.status}`);
    return data;
  }
  document.querySelectorAll("[data-follow-user]").forEach((btn) => {
    const userId = btn.getAttribute("data-follow-user");
    let following = btn.getAttribute("data-following") === "1";
    btn.addEventListener("click", async () => {
      try {
        const data = await api(`/api/social/follows/${encodeURIComponent(userId)}`, { method: following ? "DELETE" : "POST" });
        following = Boolean(data.following);
        btn.textContent = following ? "已关注" : "关注";
      } catch (err) { btn.title = err.message; }
    });
  });
})();
