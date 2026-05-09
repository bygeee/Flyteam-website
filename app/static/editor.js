async function fetchJSON(url, options = {}) {
  const headers = { ...(options.headers || {}) };
  const token = localStorage.getItem("flyteam_user_token") || localStorage.getItem("user_token") || "";
  const csrf = sessionStorage.getItem("flyteam_user_csrf") || "";
  if (token) headers["X-User-Token"] = token;
  if (csrf) headers["X-CSRF-Token"] = csrf;
  const res = await fetch(url, { credentials: "same-origin", ...options, headers });
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

function appendInlineMarkdown(parent, text) {
  const value = String(text || "");
  const regex = /(`[^`]+`|\*\*[^*]+\*\*)/g;
  let lastIndex = 0;
  let match;
  while ((match = regex.exec(value)) !== null) {
    if (match.index > lastIndex) parent.appendChild(document.createTextNode(value.slice(lastIndex, match.index)));
    const token = match[0];
    if (token.startsWith("`")) {
      parent.appendChild(createNode("code", "", token.slice(1, -1)));
    } else {
      parent.appendChild(createNode("strong", "", token.slice(2, -2)));
    }
    lastIndex = regex.lastIndex;
  }
  if (lastIndex < value.length) parent.appendChild(document.createTextNode(value.slice(lastIndex)));
}

function renderMarkdown(markdown) {
  const root = document.createDocumentFragment();
  const lines = String(markdown || "").split(/\r?\n/);
  let paragraph = [];
  let inCode = false;
  let codeLang = "";
  let codeLines = [];

  function flushParagraph() {
    const text = paragraph.join("\n").trim();
    paragraph = [];
    if (!text) return;
    const p = document.createElement("p");
    appendInlineMarkdown(p, text);
    root.appendChild(p);
  }

  function flushCode() {
    const pre = document.createElement("pre");
    const code = document.createElement("code");
    if (codeLang) code.dataset.lang = codeLang;
    code.textContent = codeLines.join("\n");
    pre.appendChild(code);
    root.appendChild(pre);
    codeLang = "";
    codeLines = [];
  }

  lines.forEach((line) => {
    const codeFence = line.match(/^```\s*(.*)$/);
    if (codeFence) {
      if (inCode) {
        flushCode();
        inCode = false;
      } else {
        flushParagraph();
        inCode = true;
        codeLang = codeFence[1].trim();
      }
      return;
    }
    if (inCode) {
      codeLines.push(line);
      return;
    }
    const trimmed = line.trim();
    if (!trimmed) {
      flushParagraph();
      return;
    }
    const heading = trimmed.match(/^(#{1,3})\s+(.+)$/);
    if (heading) {
      flushParagraph();
      const h = document.createElement(`h${Math.min(3, heading[1].length + 1)}`);
      h.textContent = heading[2];
      root.appendChild(h);
      return;
    }
    const quote = trimmed.match(/^>\s+(.+)$/);
    if (quote) {
      flushParagraph();
      const block = document.createElement("blockquote");
      appendInlineMarkdown(block, quote[1]);
      root.appendChild(block);
      return;
    }
    const image = trimmed.match(/^!\[(.*?)\]\((.*?)\)$/);
    if (image) {
      flushParagraph();
      const img = document.createElement("img");
      img.alt = image[1] || "文章图片";
      img.src = image[2] || "";
      root.appendChild(img);
      return;
    }
    paragraph.push(line);
  });

  if (inCode) flushCode();
  flushParagraph();
  return root;
}

function splitTags(value) {
  return String(value || "")
    .split(/[,，]/)
    .map((tag) => tag.trim())
    .filter(Boolean)
    .slice(0, 8);
}

function buildPayload(status) {
  return {
    title: document.getElementById("articleTitle").value.trim(),
    summary: document.getElementById("articleSummary").value.trim(),
    cover_url: document.getElementById("articleCover").value.trim(),
    category: document.getElementById("articleCategory").value.trim(),
    tags: splitTags(document.getElementById("articleTags").value),
    content_markdown: document.getElementById("articleContent").value.trim(),
    status,
  };
}

function setMessage(text) {
  const msg = document.getElementById("editorMsg");
  if (msg) msg.textContent = text || "";
}

function updatePreview() {
  const preview = document.getElementById("previewRoot");
  const content = document.getElementById("articleContent");
  const counter = document.getElementById("editorWordCount");
  if (!preview || !content) return;
  if (counter) counter.textContent = String([...content.value].length);
  preview.innerHTML = "";
  preview.appendChild(renderMarkdown(content.value));
}

let editingArticleId = "";

async function saveArticle(status) {
  setMessage(status === "published" ? "发布中..." : "保存草稿中...");
  const endpoint = editingArticleId ? `/api/blog/articles/${encodeURIComponent(editingArticleId)}` : "/api/blog/articles";
  const data = await fetchJSON(endpoint, {
    method: editingArticleId ? "PUT" : "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(buildPayload(status)),
  });
  const article = data.article || {};
  setMessage(status === "published" ? "文章已发布。" : "草稿已保存。");
  if (article.id) {
    editingArticleId = article.id;
    window.location.href = status === "published" ? `/blog/${encodeURIComponent(article.id)}` : `/editor?draft=${encodeURIComponent(article.id)}`;
  }
}

function insertAtCursor(textarea, text) {
  const start = textarea.selectionStart || 0;
  const end = textarea.selectionEnd || 0;
  textarea.value = textarea.value.slice(0, start) + text + textarea.value.slice(end);
  textarea.selectionStart = textarea.selectionEnd = start + text.length;
  textarea.focus();
  updatePreview();
}

async function uploadImage(file) {
  const body = new FormData();
  body.append("files", file);
  const data = await fetchJSON("/api/upload/blog/images", { method: "POST", body });
  const url = Array.isArray(data.saved_images) ? data.saved_images[0] : "";
  if (!url) throw new Error("图片上传失败");
  return url;
}

async function initEditor() {
  try {
    await fetchJSON("/api/users/me");
  } catch {
    window.location.href = "/user-login";
    return;
  }

  const form = document.getElementById("editorForm");
  const draftId = new URLSearchParams(location.search).get("draft") || new URLSearchParams(location.search).get("id");
  const content = document.getElementById("articleContent");
  const image = document.getElementById("articleImage");
  const insertCode = document.getElementById("insertCode");
  const saveDraft = document.getElementById("saveDraft");
  if (!form || !content || !image || !insertCode || !saveDraft) return;

  content.addEventListener("input", updatePreview);
  if (draftId) {
    try {
      const data = await fetchJSON(`/api/blog/articles/${encodeURIComponent(draftId)}`);
      const article = data.article || {};
      editingArticleId = article.id || draftId;
      document.getElementById("articleTitle").value = article.title || "";
      document.getElementById("articleSummary").value = article.summary || "";
      document.getElementById("articleCover").value = article.cover_url || "";
      document.getElementById("articleCategory").value = article.category || "";
      document.getElementById("articleTags").value = Array.isArray(article.tags) ? article.tags.join(",") : "";
      content.value = article.content_markdown || "";
      setMessage("已载入文章，可继续编辑。");
    } catch (err) {
      setMessage(err.message || "草稿加载失败");
    }
  }
  updatePreview();

  insertCode.addEventListener("click", () => {
    insertAtCursor(content, "\n```go\n// code here\n```\n");
  });

  image.addEventListener("change", async () => {
    const file = image.files && image.files[0];
    if (!file) return;
    setMessage("图片上传中...");
    try {
      const url = await uploadImage(file);
      insertAtCursor(content, `\n![文章图片](${url})\n`);
      setMessage("图片已插入正文。");
    } catch (err) {
      setMessage(err.message || "图片上传失败");
    } finally {
      image.value = "";
    }
  });

  saveDraft.addEventListener("click", async () => {
    try {
      await saveArticle("draft");
    } catch (err) {
      setMessage(err.message || "草稿保存失败");
    }
  });

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      await saveArticle("published");
    } catch (err) {
      setMessage(err.message || "文章发布失败");
    }
  });
}

initEditor();
