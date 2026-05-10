const state = {
  token: "",
  me: null,
  friends: [],
  requests: [],
  conversations: [],
  activeConversation: "",
  activeFriend: null,
  pollTimer: null,
};
const $ = (id) => document.getElementById(id);

function token() { return localStorage.getItem("flyteam_user_token") || localStorage.getItem("user_token") || ""; }
function csrf() { return sessionStorage.getItem("flyteam_user_csrf") || ""; }
function escapeHTML(s) { return String(s || "").replace(/[&<>'"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" }[c])); }
function userID(u) { return (u && (u.user_id || u.id)) || ""; }
function displayName(u) { return (u && (u.nickname || userID(u))) || "Flyteamer"; }
function firstLetter(u) { return String(displayName(u) || "F").slice(0, 1).toUpperCase(); }
function messageHTML(s) { return escapeHTML(s).replace(/\n/g, "<br>"); }
function shortText(s, n = 48) { const r = Array.from(String(s || "")); return r.length > n ? `${r.slice(0, n).join("")}…` : r.join(""); }
function formatTime(raw) {
  if (!raw) return "";
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) return raw;
  const now = new Date();
  const sameDay = d.toDateString() === now.toDateString();
  return sameDay ? d.toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" }) : d.toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" });
}
function readLocalUser() {
  try { return JSON.parse(localStorage.getItem("flyteam_user") || "null"); } catch { return null; }
}
function avatarHTML(u, cls = "friend-avatar") {
  const src = u && u.avatar_url;
  if (src) return `<span class="${cls} has-img"><img src="${escapeHTML(src)}" alt="${escapeHTML(displayName(u))}"></span>`;
  return `<span class="${cls}">${escapeHTML(firstLetter(u))}</span>`;
}
function setPeerAvatar(u) {
  const el = $("chatPeerAvatar");
  if (!el) return;
  const src = u && u.avatar_url;
  el.classList.toggle("has-img", !!src);
  el.innerHTML = src ? `<img src="${escapeHTML(src)}" alt="${escapeHTML(displayName(u))}">` : escapeHTML(firstLetter(u || { nickname: "F" }));
}

async function api(path, options = {}) {
  const headers = { ...(options.headers || {}) };
  if (state.token) headers["X-User-Token"] = state.token;
  if (csrf()) headers["X-CSRF-Token"] = csrf();
  if (options.body && !headers["Content-Type"]) headers["Content-Type"] = "application/json";
  const res = await fetch(path, { ...options, headers, credentials: "same-origin", cache: "no-store" });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.detail || `HTTP ${res.status}`);
  return data;
}

function setStatus(text) { const el = $("messageStatus"); if (el) el.textContent = text || ""; }
function setFriendStatus(text) { const el = $("friendStatus"); if (el) el.textContent = text || ""; }
function requireLoginMessage() {
  if (!state.token) {
    setStatus("请先登录普通用户账号。登录后才能私聊、发起好友申请和查看会话。");
    setFriendStatus("请先登录后再添加好友。");
    return false;
  }
  return true;
}

async function loadMe() {
  if (!state.token) return;
  try {
    const data = await api("/api/users/me");
    state.me = data.user || state.me;
    if (state.me) localStorage.setItem("flyteam_user", JSON.stringify(state.me));
  } catch {
    state.me = readLocalUser();
  }
}

async function loadFriends() {
  if (!requireLoginMessage()) return;
  try {
    const data = await api("/api/friends");
    state.friends = Array.isArray(data.items) ? data.items : [];
    renderFriends();
  } catch (err) { setFriendStatus(`好友加载失败：${err.message}`); }
}

function renderFriends() {
  const root = $("friendList");
  const count = $("friendCount");
  if (count) count.textContent = String(state.friends.length);
  if (!root) return;
  if (!state.friends.length) {
    root.innerHTML = `<div class="empty-state"><strong>还没有好友</strong><span>输入对方用户 ID 发送申请，通过后就能聊天。</span></div>`;
    return;
  }
  root.innerHTML = state.friends.map((u) => {
    const active = state.activeFriend && userID(state.activeFriend) === userID(u);
    return `<button class="friend-item contact-card ${active ? "active" : ""}" type="button" data-user="${escapeHTML(userID(u))}">
      ${avatarHTML(u)}
      <span class="contact-main"><strong>${escapeHTML(displayName(u))}</strong><em>@${escapeHTML(userID(u))}</em></span>
      <span class="contact-dot"></span>
    </button>`;
  }).join("");
  root.querySelectorAll("[data-user]").forEach((btn) => btn.addEventListener("click", () => startConversation(btn.dataset.user)));
}

async function loadRequests() {
  if (!requireLoginMessage()) return;
  try {
    const data = await api("/api/friends/requests?box=inbox");
    state.requests = Array.isArray(data.items) ? data.items : [];
    renderRequests();
  } catch (err) { setFriendStatus(`申请加载失败：${err.message}`); }
}

function renderRequests() {
  const root = $("friendRequests");
  if (!root) return;
  const pending = state.requests.filter((r) => r.status === "pending");
  if (!pending.length) {
    root.innerHTML = `<div class="empty-state small"><strong>暂无新申请</strong><span>新的好友请求会显示在这里。</span></div>`;
    return;
  }
  root.innerHTML = pending.map((r) => {
    const u = r.other_user || {};
    return `<article class="friend-request request-card" data-id="${escapeHTML(r.id)}">
      ${avatarHTML(u)}
      <div class="request-main"><strong>${escapeHTML(displayName(u))}</strong><p>${escapeHTML(r.message || "请求添加你为好友")}</p>
      <div class="friend-request-actions"><button class="community-btn primary mini-btn" data-action="accept" type="button">同意</button><button class="community-btn mini-btn" data-action="reject" type="button">拒绝</button></div></div>
    </article>`;
  }).join("");
  root.querySelectorAll("[data-action]").forEach((btn) => btn.addEventListener("click", async () => {
    const card = btn.closest("[data-id]");
    try {
      await api(`/api/friends/requests/${encodeURIComponent(card.dataset.id)}/${btn.dataset.action}`, { method: "POST" });
      await Promise.all([loadRequests(), loadFriends(), loadConversations()]);
      setFriendStatus(btn.dataset.action === "accept" ? "已同意好友申请。" : "已处理好友申请。");
    } catch (err) { setFriendStatus(`处理失败：${err.message}`); }
  }));
}

async function sendFriendRequest(event) {
  event.preventDefault();
  if (!requireLoginMessage()) return;
  const target = $("friendTarget").value.trim();
  const message = $("friendMessage").value.trim();
  if (!target) return setFriendStatus("请输入对方用户 ID。");
  try {
    await api("/api/friends/requests", { method: "POST", body: JSON.stringify({ target_user_id: target, message }) });
    $("friendTarget").value = ""; $("friendMessage").value = "";
    setFriendStatus("好友申请已发送，等待对方同意。");
  } catch (err) { setFriendStatus(`发送失败：${err.message}`); }
}

async function loadConversations() {
  if (!requireLoginMessage()) return;
  try {
    const data = await api("/api/messages/conversations");
    state.conversations = Array.isArray(data.items) ? data.items : [];
    renderConversations();
  } catch (err) { setStatus(`加载会话失败：${err.message}`); }
}

function renderConversations() {
  const root = $("conversationList");
  if (!root) return;
  if (!state.conversations.length) {
    root.innerHTML = `<div class="conversation-empty">暂无会话，点击左侧好友发起私聊</div>`;
    return;
  }
  root.innerHTML = state.conversations.map((item) => {
    const other = item.other_user || {};
    const last = item.last_message || {};
    const unread = Number(item.unread_count || 0);
    return `<button type="button" class="conversation-pill conversation-card ${item.id === state.activeConversation ? "active" : ""}" data-id="${escapeHTML(item.id)}">
      ${avatarHTML(other, "conversation-avatar")}
      <span class="conversation-main"><strong>${escapeHTML(displayName(other))}</strong><small>${escapeHTML(shortText(last.content || "还没有消息"))}</small></span>
      <span class="conversation-side"><time>${escapeHTML(formatTime(last.created_at || item.last_message_at || item.created_at))}</time>${unread ? `<em>${unread}</em>` : ""}</span>
    </button>`;
  }).join("");
  root.querySelectorAll("[data-id]").forEach((btn) => btn.addEventListener("click", () => openConversation(btn.dataset.id)));
}

async function startConversation(targetUserID) {
  try {
    const data = await api("/api/messages/conversations", { method: "POST", body: JSON.stringify({ target_user_id: targetUserID }) });
    const conv = data.conversation || {};
    await loadConversations();
    await openConversation(conv.id, conv.other_user || state.friends.find((u) => userID(u) === targetUserID) || {});
  } catch (err) { setStatus(`无法发起私聊：${err.message}`); }
}

async function openConversation(id, other = null) {
  if (!id) return;
  state.activeConversation = id;
  const conv = state.conversations.find((c) => c.id === id);
  const user = other || (conv && conv.other_user) || {};
  state.activeFriend = user;
  const title = $("chatTitle");
  const subtitle = $("chatSubtitle");
  if (title) title.textContent = user.nickname ? `与 ${user.nickname} 的私聊` : "私聊会话";
  if (subtitle) subtitle.textContent = userID(user) ? `@${userID(user)} · 数据库存储 · 好友会话` : "好友消息";
  setPeerAvatar(user);
  renderFriends();
  renderConversations();
  try {
    const data = await api(`/api/messages/conversations/${encodeURIComponent(id)}/messages`);
    renderMessages(Array.isArray(data.items) ? data.items : []);
    loadConversations();
  } catch (err) { setStatus(`加载消息失败：${err.message}`); }
}

function renderMessages(items) {
  const root = $("messageBox");
  if (!root) return;
  if (!items.length) {
    root.innerHTML = `<div class="empty-chat-illustration"><div>💬</div><strong>暂无消息</strong><span>发出第一句问候吧。</span></div>`;
    return;
  }
  root.innerHTML = items.map((msg) => {
    const sender = msg.sender || {};
    const mine = !!msg.mine;
    return `<article class="chat-row ${mine ? "mine" : ""}">
      ${avatarHTML(sender, "chat-avatar")}
      <div class="chat-bubble-wrap">
        <div class="chat-meta"><strong>${mine ? "我" : escapeHTML(displayName(sender))}</strong><time>${escapeHTML(formatTime(msg.created_at))}</time></div>
        <div class="message-bubble">${messageHTML(msg.content || "")}</div>
      </div>
    </article>`;
  }).join("");
  root.scrollTop = root.scrollHeight;
}

async function sendMessage() {
  const text = $("messageText");
  if (!state.activeConversation) return setStatus("请先从好友列表选择一个好友。");
  const content = text.value.trim();
  if (!content) return;
  try {
    await api(`/api/messages/conversations/${encodeURIComponent(state.activeConversation)}/messages`, { method: "POST", body: JSON.stringify({ content }) });
    text.value = "";
    await openConversation(state.activeConversation, state.activeFriend);
  } catch (err) { setStatus(`发送失败：${err.message}`); }
}

function initPolling() {
  if (state.pollTimer) clearInterval(state.pollTimer);
  state.pollTimer = setInterval(() => {
    if (!state.token) return;
    loadConversations();
    if (state.activeConversation) openConversation(state.activeConversation, state.activeFriend);
  }, 15000);
}

async function init() {
  state.token = token();
  state.me = readLocalUser();
  $("friendRequestForm")?.addEventListener("submit", sendFriendRequest);
  $("refreshRequests")?.addEventListener("click", () => { loadRequests(); loadFriends(); });
  $("refreshConversations")?.addEventListener("click", loadConversations);
  $("messageForm")?.addEventListener("submit", (event) => { event.preventDefault(); sendMessage(); });
  $("messageText")?.addEventListener("keydown", (event) => { if (event.key === "Enter" && !event.shiftKey) { event.preventDefault(); sendMessage(); } });
  if (!state.token) { requireLoginMessage(); return; }
  await loadMe();
  await Promise.all([loadFriends(), loadRequests(), loadConversations()]);
  initPolling();
}
init();
