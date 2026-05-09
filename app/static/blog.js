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

function formatDate(value) {
  if (!value) return "未发布";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleDateString("zh-CN");
}

function renderArticleCard(article) {
  const card = document.createElement("a");
  card.className = "article-card";
  card.href = `/blog/${encodeURIComponent(article.id)}`;

  card.appendChild(createNode("h3", "", article.title || "未命名文章"));
  card.appendChild(createNode("p", "", article.summary || "暂无摘要。"));

  const meta = createNode("div", "article-meta");
  meta.appendChild(createNode("span", "", article.author_nickname || article.author_user_id || "匿名作者"));
  meta.appendChild(createNode("span", "", formatDate(article.published_at || article.created_at)));
  meta.appendChild(createNode("span", "", `${article.views || 0} 次浏览`));
  card.appendChild(meta);

  if (Array.isArray(article.tags) && article.tags.length) {
    const tags = createNode("div", "article-tags");
    article.tags.forEach((tag) => tags.appendChild(createNode("span", "article-tag", tag)));
    card.appendChild(tags);
  }

  return card;
}

async function initBlogList() {
  const list = document.getElementById("blogList");
  const status = document.getElementById("blogStatus");
  if (!list || !status) return;

  try {
    const data = await fetchJSON("/api/blog/articles");
    const articles = Array.isArray(data.articles) ? data.articles : [];
    list.innerHTML = "";
    if (!articles.length) {
      status.textContent = "暂时还没有公开文章。";
      return;
    }
    status.textContent = `共 ${articles.length} 篇公开文章`;
    articles.forEach((article) => list.appendChild(renderArticleCard(article)));
  } catch (err) {
    status.textContent = err.message || "文章加载失败";
  }
}

initBlogList();
