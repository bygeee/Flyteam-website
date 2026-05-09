const state = { token: "", conversations: [], activeConversation: "" };
const $ = (id) => document.getElementById(id);

function loadToken() {
  const saved = localStorage.getItem("flyteam_user_token") || localStorage.getItem("user_token") || "";
  const input = $("userToken");
  if (input) {
    input.value = saved;
    input.addEventListener("change", () => {
      state.token = input.value.trim();
      localStorage.setItem("flyteam_user_token", state.token);
      loadConversations();
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

function setStatus(text) {
  const el = $("messageStatus");
  if (el) el.textContent = text || "";
}

function renderConversations() {
  const root = $("conversationList");
  if (!root) return;
  root.innerHTML = "";
  if (!state.conversations.length) {
    root.innerHTML = '<p class="community-muted">暂无会话。</p>';
    return;
  }
  state.conversations.forEach((item) => {
    const div = document.createElement("div");
    div.className = "community-item" + (item.id === state.activeConversation ? " active" : "");
    const other = item.other_user || {};
    div.innerHTML = `<strong>${escapeHTML(other.nickname || other.id || "未知用户")}</strong><br><span class="community-muted">未读 ${Number(item.unread_count || 0)} · ${escapeHTML((item.last_message || {}).content || "暂无消息")}</span>`;
    div.addEventListener("click", () => openConversation(item.id, other));
    root.appendChild(div);
  });
}

async function loadConversations() {
  if (!state.token) {
    setStatus("请先填写用户 Token。后续普通用户登录完成后会自动带上。 ");
    return;
  }
  try {
    const data = await api("/api/messages/conversations");
    state.conversations = Array.isArray(data.items) ? data.items : [];
    renderConversations();
  } catch (err) {
    setStatus(`加载会话失败：${err.message}`);
  }
}

async function openConversation(id, other = {}) {
  state.activeConversation = id;
  renderConversations();
  const title = $("chatTitle");
  if (title) title.textContent = other.nickname ? `与 ${other.nickname} 的私信` : "私信会话";
  try {
    const data = await api(`/api/messages/conversations/${encodeURIComponent(id)}/messages`);
    renderMessages(Array.isArray(data.items) ? data.items : []);
    loadConversations();
  } catch (err) {
    setStatus(`加载消息失败：${err.message}`);
  }
}

function renderMessages(items) {
  const root = $("messageBox");
  if (!root) return;
  root.innerHTML = "";
  items.forEach((msg) => {
    const div = document.createElement("div");
    div.className = "message-bubble" + (msg.mine ? " mine" : "");
    div.textContent = msg.content || "";
    root.appendChild(div);
  });
  root.scrollTop = root.scrollHeight;
}

function escapeHTML(s) {
  return String(s || "").replace(/[&<>'"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" }[c]));
}

function initForms() {
  const newForm = $("newConversation");
  newForm?.addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      const target = $("targetUser").value.trim();
      const data = await api("/api/messages/conversations", { method: "POST", body: JSON.stringify({ target_user_id: target }) });
      $("targetUser").value = "";
      await loadConversations();
      if (data.conversation) openConversation(data.conversation.id, data.conversation.other_user || {});
    } catch (err) {
      setStatus(`创建会话失败：${err.message}`);
    }
  });

  const form = $("messageForm");
  const text = $("messageText");
  const send = async () => {
    if (!state.activeConversation) return setStatus("请先选择会话。");
    const content = text.value.trim();
    if (!content) return;
    try {
      await api(`/api/messages/conversations/${encodeURIComponent(state.activeConversation)}/messages`, { method: "POST", body: JSON.stringify({ content }) });
      text.value = "";
      await openConversation(state.activeConversation);
    } catch (err) {
      setStatus(`发送失败：${err.message}`);
    }
  };
  form?.addEventListener("submit", (event) => { event.preventDefault(); send(); });
  text?.addEventListener("keydown", (event) => {
    if (event.key === "Enter" && !event.shiftKey) { event.preventDefault(); send(); }
  });
}

loadToken();
initForms();
loadConversations();
