const state = { token: "", groups: [], activeGroup: "" };
const $ = (id) => document.getElementById(id);

function loadToken() {
  const saved = localStorage.getItem("flyteam_user_token") || localStorage.getItem("user_token") || "";
  const input = $("userToken");
  if (input) {
    input.value = saved;
    input.addEventListener("change", () => {
      state.token = input.value.trim();
      localStorage.setItem("flyteam_user_token", state.token);
      loadGroups();
    });
  }
  state.token = saved;
}

async function api(path, options = {}) {
  const headers = { ...(options.headers || {}) };
  if (state.token) headers["X-User-Token"] = state.token;
  if (options.body && !headers["Content-Type"]) headers["Content-Type"] = "application/json";
  const res = await fetch(path, { ...options, headers });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.detail || `HTTP ${res.status}`);
  return data;
}

function setStatus(text) { const el = $("groupStatus"); if (el) el.textContent = text || ""; }
function escapeHTML(s) { return String(s || "").replace(/[&<>'"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" }[c])); }

function renderGroups() {
  const root = $("groupList");
  if (!root) return;
  root.innerHTML = "";
  if (!state.groups.length) { root.innerHTML = '<p class="community-muted">暂无公开群。</p>'; return; }
  state.groups.forEach((group) => {
    const div = document.createElement("div");
    div.className = "community-item" + (group.id === state.activeGroup ? " active" : "");
    div.innerHTML = `<strong>${escapeHTML(group.name)}</strong><br><span class="community-muted">${escapeHTML(group.visibility)} · ${Number(group.member_count || 0)} 人</span>`;
    div.addEventListener("click", () => openGroup(group.id));
    root.appendChild(div);
  });
}

async function loadGroups() {
  try {
    const data = await api("/api/groups");
    state.groups = Array.isArray(data.items) ? data.items : [];
    renderGroups();
  } catch (err) {
    setStatus(`加载群失败：${err.message}`);
  }
}

async function openGroup(id) {
  state.activeGroup = id;
  renderGroups();
  try {
    const data = await api(`/api/groups/${encodeURIComponent(id)}`);
    const group = data.group || {};
    const title = $("groupTitle");
    if (title) title.textContent = group.name || "群聊";
    if (group.my_status === "active") await loadMessages(id);
    else renderMessages([]);
  } catch (err) { setStatus(`打开群失败：${err.message}`); }
}

async function loadMessages(id) {
  const data = await api(`/api/groups/${encodeURIComponent(id)}/messages`);
  renderMessages(Array.isArray(data.items) ? data.items : []);
}

function renderMessages(items) {
  const root = $("groupMessages");
  if (!root) return;
  root.innerHTML = "";
  items.forEach((msg) => {
    const div = document.createElement("div");
    div.className = "message-bubble";
    const sender = msg.sender || {};
    div.textContent = `${sender.nickname || sender.id || "成员"}：${msg.content || ""}`;
    root.appendChild(div);
  });
  root.scrollTop = root.scrollHeight;
}

function initForms() {
  $("newGroup")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      const name = $("groupName").value.trim();
      const data = await api("/api/groups", { method: "POST", body: JSON.stringify({ name, visibility: "public" }) });
      $("groupName").value = "";
      await loadGroups();
      if (data.group) openGroup(data.group.id);
    } catch (err) { setStatus(`创建群失败：${err.message}`); }
  });

  $("joinGroup")?.addEventListener("click", async () => {
    if (!state.activeGroup) return setStatus("请先选择群。");
    try { await api(`/api/groups/${encodeURIComponent(state.activeGroup)}/members`, { method: "POST", body: "{}" }); await openGroup(state.activeGroup); }
    catch (err) { setStatus(`加群失败：${err.message}`); }
  });

  const form = $("groupMessageForm");
  const text = $("groupMessageText");
  const send = async () => {
    if (!state.activeGroup) return setStatus("请先选择群。");
    const content = text.value.trim();
    if (!content) return;
    try {
      await api(`/api/groups/${encodeURIComponent(state.activeGroup)}/messages`, { method: "POST", body: JSON.stringify({ content }) });
      text.value = "";
      await loadMessages(state.activeGroup);
    } catch (err) { setStatus(`发送失败：${err.message}`); }
  };
  form?.addEventListener("submit", (event) => { event.preventDefault(); send(); });
  text?.addEventListener("keydown", (event) => { if (event.key === "Enter" && !event.shiftKey) { event.preventDefault(); send(); } });
}

loadToken();
initForms();
loadGroups();
