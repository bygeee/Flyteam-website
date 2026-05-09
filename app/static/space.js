async function fetchJSON(url, options = {}) {
  const res = await fetch(url, { credentials: "same-origin", ...options });
  const raw = await res.text();
  let data = null;
  try {
    data = raw ? JSON.parse(raw) : null;
  } catch {
    data = null;
  }
  if (!res.ok) throw new Error((data && data.detail) || raw || `HTTP ${res.status}`);
  return data || {};
}

function createNode(tag, className, text) {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}

function profileIDFromPath() {
  const parts = window.location.pathname.split("/").filter(Boolean);
  return parts.length >= 2 && parts[0] === "space" ? decodeURIComponent(parts[1]) : "";
}

function renderAvatar(user) {
  if (user.avatar_url) {
    const img = document.createElement("img");
    img.className = "profile-avatar";
    img.src = user.avatar_url;
    img.alt = user.nickname || user.user_id || "用户头像";
    return img;
  }
  const fallback = createNode("div", "profile-avatar profile-avatar-fallback", (user.nickname || user.user_id || "U").slice(0, 1).toUpperCase());
  return fallback;
}

function appendStat(parent, label, value) {
  const box = createNode("div", "profile-stat");
  box.appendChild(createNode("strong", "", String(value || 0)));
  box.appendChild(createNode("span", "", label));
  parent.appendChild(box);
}

function renderProfile(data) {
  const root = document.getElementById("profileRoot");
  if (!root) return;
  const user = data.user || {};
  const stats = data.stats || {};
  root.innerHTML = "";

  const head = createNode("div", "profile-head");
  head.appendChild(renderAvatar(user));

  const main = createNode("div", "profile-main");
  main.appendChild(createNode("h1", "", user.nickname || user.user_id || "用户主页"));
  main.appendChild(createNode("p", "community-muted", `@${user.user_id || "unknown"}`));
  main.appendChild(createNode("p", "", user.bio || "这个用户还没有填写简介。"));
  head.appendChild(main);
  root.appendChild(head);

  const statRow = createNode("div", "profile-stats");
  appendStat(statRow, "文章", stats.articles);
  appendStat(statRow, "粉丝", stats.followers);
  appendStat(statRow, "关注", stats.following);
  root.appendChild(statRow);

  if (data.is_owner) root.appendChild(renderEditForm(user));
}

function renderEditForm(user) {
  const wrap = createNode("section", "profile-edit");
  wrap.appendChild(createNode("h2", "", "编辑资料"));

  const form = document.createElement("form");
  form.id = "profileEditForm";

  const nickname = document.createElement("input");
  nickname.id = "profileNickname";
  nickname.placeholder = "昵称";
  nickname.value = user.nickname || "";
  nickname.required = true;

  const avatar = document.createElement("input");
  avatar.id = "profileAvatar";
  avatar.placeholder = "头像 URL";
  avatar.value = user.avatar_url || "";

  const bio = document.createElement("textarea");
  bio.id = "profileBio";
  bio.placeholder = "个人简介，最多 300 字";
  bio.rows = 5;
  bio.value = user.bio || "";

  const actions = createNode("div", "profile-actions");
  const button = document.createElement("button");
  button.type = "submit";
  button.className = "community-button";
  button.textContent = "保存资料";
  const msg = createNode("span", "community-muted");
  msg.id = "profileEditMsg";
  actions.appendChild(button);
  actions.appendChild(msg);

  form.appendChild(nickname);
  form.appendChild(avatar);
  form.appendChild(bio);
  form.appendChild(actions);
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    msg.textContent = "保存中...";
    try {
      const id = profileIDFromPath();
      const next = await fetchJSON(`/api/users/${encodeURIComponent(id)}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          nickname: nickname.value.trim(),
          avatar_url: avatar.value.trim(),
          bio: bio.value.trim(),
        }),
      });
      msg.textContent = "已保存";
      renderProfile(next);
    } catch (err) {
      msg.textContent = err.message || "保存失败";
    }
  });

  wrap.appendChild(form);
  return wrap;
}

async function initSpace() {
  const root = document.getElementById("profileRoot");
  if (!root) return;
  const id = profileIDFromPath();
  if (!id) {
    root.textContent = "缺少用户 ID。";
    return;
  }
  try {
    const data = await fetchJSON(`/api/users/${encodeURIComponent(id)}`);
    if (data.user && data.user.nickname) document.title = `${data.user.nickname} - Flyteam 用户主页`;
    renderProfile(data);
  } catch (err) {
    root.textContent = err.message || "用户主页加载失败。";
  }
}

initSpace();
