const state = { token: "", me: null, groups: [], friends: [], activeGroup: "", activeGroupData: null, pollTimer: null };
const $ = (id) => document.getElementById(id);
function token() { return localStorage.getItem("flyteam_user_token") || localStorage.getItem("user_token") || ""; }
function csrf() { return sessionStorage.getItem("flyteam_user_csrf") || ""; }
function escapeHTML(s) { return String(s || "").replace(/[&<>'"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" }[c])); }
function userID(u) { return (u && (u.user_id || u.id)) || ""; }
function displayName(u) { return (u && (u.nickname || userID(u))) || "Flyteamer"; }
function firstLetter(u) { return String(displayName(u) || "F").slice(0,1).toUpperCase(); }
function messageHTML(s) { return escapeHTML(s).replace(/\n/g, "<br>"); }
function formatTime(raw) {
  if (!raw) return "";
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) return raw;
  const now = new Date();
  const sameDay = d.toDateString() === now.toDateString();
  return sameDay ? d.toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" }) : d.toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" });
}
function readLocalUser() { try { return JSON.parse(localStorage.getItem("flyteam_user") || "null"); } catch { return null; } }
function avatarHTML(u, cls = "friend-avatar") {
  const src = u && u.avatar_url;
  if (src) return `<span class="${cls} has-img"><img src="${escapeHTML(src)}" alt="${escapeHTML(displayName(u))}"></span>`;
  return `<span class="${cls}">${escapeHTML(firstLetter(u))}</span>`;
}
function groupAvatarHTML(group, cls = "friend-avatar") {
  const src = group && group.avatar_url;
  if (src) return `<span class="${cls} has-img"><img src="${escapeHTML(src)}" alt="${escapeHTML(group.name || "群聊")}"></span>`;
  return `<span class="${cls} group-mark">群</span>`;
}
function setGroupAvatar(group) {
  const el = $("groupAvatar");
  if (!el) return;
  const src = group && group.avatar_url;
  el.classList.toggle("has-img", !!src);
  el.innerHTML = src ? `<img src="${escapeHTML(src)}" alt="${escapeHTML(group.name || "群聊")}">` : "群";
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
function setStatus(text) { const el = $("groupStatus"); if (el) el.textContent = text || ""; }
function requireLogin() { if (!state.token) { setStatus("请先登录普通用户账号。登录后才能建群、邀请好友和发送消息。"); return false; } return true; }
async function loadMe() {
  if (!state.token) return;
  try {
    const data = await api("/api/users/me");
    state.me = data.user || state.me;
    if (state.me) localStorage.setItem("flyteam_user", JSON.stringify(state.me));
  } catch { state.me = readLocalUser(); }
}

async function loadFriends() {
  if (!requireLogin()) return;
  try { const data = await api("/api/friends"); state.friends = Array.isArray(data.items) ? data.items : []; renderFriendChecks(); }
  catch (err) { setStatus(`好友加载失败：${err.message}`); }
}
function renderFriendChecks() {
  for (const id of ["createFriendChecks", "inviteFriendChecks"]) {
    const root = $(id); if (!root) continue;
    if (!state.friends.length) { root.innerHTML = `<div class="empty-state small"><strong>暂无好友</strong><span>先去私信页添加好友，再拉人建群。</span></div>`; continue; }
    root.innerHTML = state.friends.map((u) => `<label class="friend-check selector-card"><input type="checkbox" value="${escapeHTML(userID(u))}">${avatarHTML(u)}<span><strong>${escapeHTML(displayName(u))}</strong><em>@${escapeHTML(userID(u))}</em></span></label>`).join("");
  }
}
function checkedValues(rootID) { return [...document.querySelectorAll(`#${rootID} input[type="checkbox"]:checked`)].map((i) => i.value); }
function clearChecked(rootID) { document.querySelectorAll(`#${rootID} input[type="checkbox"]`).forEach((i) => { i.checked = false; }); }

async function loadGroups() {
  if (!requireLogin()) return;
  try { const data = await api("/api/groups"); state.groups = Array.isArray(data.items) ? data.items : []; renderGroups(); }
  catch (err) { setStatus(`加载群失败：${err.message}`); }
}
function renderGroups() {
  const root = $("groupList"); if (!root) return;
  if (!state.groups.length) { root.innerHTML = `<div class="empty-state"><strong>暂无群聊</strong><span>创建一个方向群，把好友拉进来讨论。</span></div>`; return; }
  root.innerHTML = state.groups.map((group) => `<button class="friend-item contact-card group-card ${group.id === state.activeGroup ? "active" : ""}" type="button" data-id="${escapeHTML(group.id)}">
    ${groupAvatarHTML(group)}
    <span class="contact-main"><strong>${escapeHTML(group.name)}</strong><em>${Number(group.member_count || 0)} 人 · ${group.visibility === "private" ? "私密群" : "公开群"}</em></span>
    <span class="contact-dot"></span>
  </button>`).join("");
  root.querySelectorAll("[data-id]").forEach((btn) => btn.addEventListener("click", () => openGroup(btn.dataset.id)));
}

async function openGroup(id) {
  if (!id) return;
  state.activeGroup = id; renderGroups();
  try {
    const data = await api(`/api/groups/${encodeURIComponent(id)}`);
    const group = data.group || {};
    state.activeGroupData = group;
    $("groupTitle").textContent = group.name || "群聊";
    $("groupSubtitle").textContent = `${Number(group.member_count || 0)} 位成员 · ${group.visibility === "private" ? "私密群" : "公开群"} · 聊天记录数据库保存`;
    setGroupAvatar(group);
    await loadMessages(id);
  } catch (err) { setStatus(`打开群失败：${err.message}`); }
}
async function loadMessages(id) {
  const data = await api(`/api/groups/${encodeURIComponent(id)}/messages`);
  renderMessages(Array.isArray(data.items) ? data.items : []);
}
function renderMessages(items) {
  const root = $("groupMessages"); if (!root) return;
  if (!items.length) { root.innerHTML = `<div class="empty-chat-illustration"><div>👥</div><strong>暂无群消息</strong><span>开启第一次方向讨论。</span></div>`; return; }
  const meID = userID(state.me || {});
  root.innerHTML = items.map((msg) => {
    const s = msg.sender || {};
    const mine = !!msg.mine || (meID && userID(s) === meID);
    return `<article class="chat-row ${mine ? "mine" : ""}">
      ${avatarHTML(s, "chat-avatar")}
      <div class="chat-bubble-wrap">
        <div class="chat-meta"><strong>${mine ? "我" : escapeHTML(displayName(s))}</strong><time>${escapeHTML(formatTime(msg.created_at))}</time></div>
        <div class="message-bubble">${messageHTML(msg.content || "")}</div>
      </div>
    </article>`;
  }).join("");
  root.scrollTop = root.scrollHeight;
}
async function createGroup(event) {
  event.preventDefault(); if (!requireLogin()) return;
  const name = $("groupName").value.trim(); const intro = $("groupIntro").value.trim();
  if (!name) return setStatus("请输入群名称。");
  try {
    const data = await api("/api/groups", { method: "POST", body: JSON.stringify({ name, intro, visibility: "private", member_user_ids: checkedValues("createFriendChecks") }) });
    $("groupName").value = ""; $("groupIntro").value = ""; clearChecked("createFriendChecks");
    await loadGroups(); if (data.group) await openGroup(data.group.id);
    setStatus("群聊已创建，选中的好友已加入。")
  } catch (err) { setStatus(`创建群失败：${err.message}`); }
}
async function inviteFriends() {
  if (!state.activeGroup) return setStatus("请先选择一个群。");
  const ids = checkedValues("inviteFriendChecks"); if (!ids.length) return setStatus("请先勾选要邀请的好友。");
  try {
    for (const id of ids) await api(`/api/groups/${encodeURIComponent(state.activeGroup)}/members`, { method: "POST", body: JSON.stringify({ user_id: id }) });
    clearChecked("inviteFriendChecks");
    await openGroup(state.activeGroup); setStatus("已邀请选中的好友进群。");
  } catch (err) { setStatus(`邀请失败：${err.message}`); }
}
async function sendGroupMessage() {
  if (!state.activeGroup) return setStatus("请先选择群。");
  const text = $("groupMessageText"); const content = text.value.trim(); if (!content) return;
  try { await api(`/api/groups/${encodeURIComponent(state.activeGroup)}/messages`, { method: "POST", body: JSON.stringify({ content }) }); text.value = ""; await loadMessages(state.activeGroup); }
  catch (err) { setStatus(`发送失败：${err.message}`); }
}
function initPolling() {
  if (state.pollTimer) clearInterval(state.pollTimer);
  state.pollTimer = setInterval(() => {
    if (!state.token) return;
    loadGroups();
    if (state.activeGroup) loadMessages(state.activeGroup);
  }, 15000);
}
async function init() {
  state.token = token();
  state.me = readLocalUser();
  $("newGroup")?.addEventListener("submit", createGroup);
  $("refreshGroups")?.addEventListener("click", loadGroups);
  $("inviteFriends")?.addEventListener("click", inviteFriends);
  $("groupMessageForm")?.addEventListener("submit", (event) => { event.preventDefault(); sendGroupMessage(); });
  $("groupMessageText")?.addEventListener("keydown", (event) => { if (event.key === "Enter" && !event.shiftKey) { event.preventDefault(); sendGroupMessage(); } });
  if (!state.token) { requireLogin(); return; }
  await loadMe();
  await Promise.all([loadFriends(), loadGroups()]);
  initPolling();
}
init();
