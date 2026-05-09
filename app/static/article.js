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

function articleIDFromPath() {
  const parts = window.location.pathname.split("/").filter(Boolean);
  return parts.length >= 2 && parts[0] === "blog" ? decodeURIComponent(parts[1]) : "";
}

function formatDate(value) {
  if (!value) return "未发布";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN");
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
  const root = createNode("div", "article-body");
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
      const level = String(Math.min(3, heading[1].length + 1));
      const h = document.createElement(`h${level}`);
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

async function initArticle() {
  const root = document.getElementById("articleRoot");
  if (!root) return;
  const id = articleIDFromPath();
  if (!id) {
    root.textContent = "缺少文章 ID。";
    return;
  }

  try {
    const data = await fetchJSON(`/api/blog/articles/${encodeURIComponent(id)}`);
    const article = data.article || {};
    document.title = `${article.title || "文章详情"} - Flyteam`;
    root.innerHTML = "";

    const meta = createNode("div", "article-meta");
    meta.appendChild(createNode("span", "", article.author_nickname || article.author_user_id || "匿名作者"));
    meta.appendChild(createNode("span", "", formatDate(article.published_at || article.created_at)));
    meta.appendChild(createNode("span", "", `${article.views || 0} 次浏览`));

    root.appendChild(createNode("h1", "", article.title || "未命名文章"));
    root.appendChild(meta);

    if (Array.isArray(article.tags) && article.tags.length) {
      const tags = createNode("div", "article-tags");
      article.tags.forEach((tag) => tags.appendChild(createNode("span", "article-tag", tag)));
      root.appendChild(tags);
    }

    if (article.summary) root.appendChild(createNode("p", "community-muted", article.summary));
    root.appendChild(renderMarkdown(article.content_markdown || ""));

    fetchJSON(`/api/blog/articles/${encodeURIComponent(id)}/view`, { method: "POST" }).catch(() => {});
  } catch (err) {
    root.textContent = err.message || "文章加载失败。";
  }
}

initArticle();
