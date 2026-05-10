# Flyteam Website 五人协作分配文档

> 项目成员代号：`z3`、`grand`、`dong`、`dl`、`wang`  
> 后端成员：`z3`、`grand`  
> 前端成员：`dong`、`dl`、`wang`  
> 项目目标：在保留当前 Flyteam 官网、管理后台、RAG、博客社区等全部已有功能的基础上，继续完成架构重构、前后端体验升级、CI/CD 自动化部署和后续功能扩展。  
> 最高原则：任何人不得删除或破坏 VPS 运行数据、上传缓存、数据库、RAG 知识库和线上已可用功能。

---

## 1. 总体分工

| 成员 | 方向 | 定位 | 主要职责 |
| --- | --- | --- | --- |
| `z3` | 后端 | 项目负责人 / 后端架构 / 最终合并与发布 | 总体架构、数据库方案、权限安全、CI/CD、VPS 部署、代码审核、最终合并、线上故障兜底 |
| `grand` | 后端 | 社区与业务后端负责人 | 用户系统、博客文章、评论互动、关注私信、群聊通知、RAG API、业务测试 |
| `dong` | 前端 | 官网宣传前台负责人 | 首页、团队简介、新闻、回顾、奖项、Flyteamers、招新页面、整体视觉和响应式 |
| `dl` | 前端 | 博客和用户中心负责人 | 博客广场、文章详情、编辑器、登录注册、个人主页、个人中心、文章交互体验 |
| `wang` | 前端 | 管理后台和聊天体验负责人 | 管理员后台、用户审核、报名管理、聊天/私信/群聊页面、通知中心、后台可视化 |

---

## 2. 不能破坏的现有功能

所有人开发前必须确认：以下功能必须继续可用。

- 首页全屏照片墙、随机背景、图片轮播。
- 团队新闻、团队回顾、回顾相册。
- 奖项荣誉分类展示。
- Flyteamers / 前辈墙 / 帮主 / 负责人 / 团队管理分类。
- 招新报名、C 语言动态验证码、报名管理。
- `/admin` 管理后台、管理员登录、超级管理员权限。
- 普通用户注册审核、博客、评论、点赞、收藏、关注、私信、群聊、通知。
- 文件上传安全限制、头像上传、新闻/奖项/回顾/前辈图片上传。
- RAG 知识库上传、重建、问答。
- VPS 上已有 `storage/`、`storage/uploads/`、`storage/flyteam.db`、`.env` 等运行数据。

以下目录和文件默认不允许直接删除：

```text
storage/
storage/uploads/
storage/flyteam.db
storage/*.json
storage/chroma/
storage/rag_index_go.json
.env
```

如确实需要迁移或清理，必须先由 `z3` 确认并备份。

---

## 3. 当前代码目录所有权

### 3.1 z3：后端架构、安全、部署

主要负责：

```text
.github/workflows/
docs/CI_CD.md
scripts/
cmd/flyteam-server/main.go
cmd/flyteam-server/internal/app/server.go
cmd/flyteam-server/internal/app/http_*.go
cmd/flyteam-server/internal/app/routes_admin_backend.go
cmd/flyteam-server/internal/app/routes_system_api.go
cmd/flyteam-server/internal/app/admin_*.go
cmd/flyteam-server/internal/app/system_*.go
cmd/flyteam-server/internal/database/
```

重点任务：

- 维护后端启动、配置加载、路由总入口。
- 维护管理员后台、超级管理员、权限边界、CSRF、上传安全。
- 维护 SQLite 数据库初始化、迁移兼容、旧 JSON 数据读取。
- 维护 GitHub Actions CI/CD 和 VPS 部署流程。
- 审核所有 PR，负责合并到 `main` 或后续 `develop`。
- 每次合并后做功能回归、安全检查和部署检查。

推荐分支：

```text
feature/z3-backend-foundation
feature/z3-security-hardening
feature/z3-ci-cd
feature/z3-deploy-fix
feature/z3-db-migration
```

---

### 3.2 grand：后端用户、博客、社区、RAG

主要负责：

```text
cmd/flyteam-server/internal/app/routes_user_api.go
cmd/flyteam-server/internal/app/user_*.go
cmd/flyteam-server/internal/app/routes_recruit.go       # 涉及报名业务时与 z3 对齐
cmd/flyteam-server/internal/app/system_rag.go           # RAG 业务部分与 z3 对齐
cmd/flyteam-server/internal/blog/
```

重点任务：

- 普通用户注册、登录、审核状态、资料修改。
- 博客文章发布、编辑、草稿、浏览量、推荐排序。
- 评论、点赞、收藏、关注、好友申请。
- 私信、群聊、通知、搜索 API。
- RAG 问答后端能力，后续负责流式输出 API。
- 为新增 API 补齐单元测试和接口文档。

推荐分支：

```text
feature/grand-user-auth-api
feature/grand-blog-api
feature/grand-social-message-api
feature/grand-group-notification-api
feature/grand-rag-stream-api
```

---

### 3.3 dong：前端官网宣传站

主要负责：

```text
app/static/pages/index.html
app/static/pages/intro.html
app/static/pages/news.html
app/static/pages/awards.html
app/static/pages/review.html
app/static/pages/review_detail.html
app/static/pages/flyteamers.html
app/static/pages/recruit.html
app/static/js/public.js
app/static/js/news.js
app/static/js/flyteamers.js
app/static/js/recruit.js
app/static/js/page_backdrop.js
app/static/css/public.css
app/static/css/styles.css
```

重点任务：

- 首页视觉、照片墙、随机背景、滚动动画。
- 团队简介、团队新闻、团队回顾、奖项荣誉、前辈墙页面体验。
- 招新报名表单和验证码交互体验。
- 所有公开官网页面的移动端适配。
- 保证图片双击放大、占位图、玻璃卡片、随机背景等视觉逻辑统一。

推荐分支：

```text
feature/dong-home-polish
feature/dong-public-pages
feature/dong-recruit-ui
feature/dong-mobile-responsive
```

---

### 3.4 dl：前端博客和用户中心

主要负责：

```text
app/static/pages/blog.html
app/static/pages/article.html
app/static/pages/editor.html
app/static/pages/account.html
app/static/pages/space.html
app/static/pages/user_login.html
app/static/pages/user_register.html
app/static/js/blog.js
app/static/js/article.js
app/static/js/editor.js
app/static/js/account.js
app/static/js/space.js
app/static/js/user_login.js
app/static/js/user_register.js
app/static/css/community.css
```

重点任务：

- 博客广场排版、文章卡片、推荐区、热门区。
- 文章详情页阅读体验、代码块、图片、评论区入口。
- 富文本/Markdown 编辑器体验。
- 用户登录、注册、审核状态提示。
- 个人主页、个人中心、头像修改、资料修改。
- 未登录只能看公开内容，评论/关注/私信等操作必须提示登录。

推荐分支：

```text
feature/dl-blog-square
feature/dl-article-detail
feature/dl-editor-ui
feature/dl-user-center
feature/dl-login-register
```

---

### 3.5 wang：前端管理后台、聊天、通知

主要负责：

```text
app/static/pages/admin.html
app/static/pages/login.html
app/static/pages/messages.html
app/static/pages/groups.html
app/static/js/app.js
app/static/js/login.js
app/static/js/messages.js
app/static/js/groups.js
app/static/css/community.css
app/static/css/styles.css        # 后台/聊天公共样式需要和 dong、dl 对齐
```

重点任务：

- 管理员登录页和 `/admin` 后台体验。
- 后台用户审核红点、未审核优先展示、状态标记。
- 后台内容管理：新闻、回顾、奖项、前辈、报名、RAG 管理交互。
- 私信页面、好友列表、群聊页面，体验参考 QQ / WeChat。
- 通知中心、未读状态、空状态、错误提示。
- 后台页面的移动端基础可用性。

推荐分支：

```text
feature/wang-admin-dashboard
feature/wang-user-review-admin
feature/wang-message-ui
feature/wang-group-ui
feature/wang-notification-ui
```

---

## 4. 后续 Vue 重构时的目录所有权

如果项目进入 Vue 3 + Vite 重构阶段，按下面分配：

| 成员 | 未来目录所有权 |
| --- | --- |
| `z3` | `backend/cmd/`、`backend/internal/config/`、`backend/internal/middleware/`、`backend/internal/database/`、`deploy/`、`.github/workflows/` |
| `grand` | `backend/internal/api/v1/user*`、`blog*`、`social*`、`message*`、`group*`、`notification*`、`rag*`，以及对应 service/repository |
| `dong` | `frontend/src/views/public/`、`frontend/src/views/recruit/`、`frontend/src/components/public/` |
| `dl` | `frontend/src/views/blog/`、`frontend/src/views/user/`、`frontend/src/views/editor/`、`frontend/src/components/blog/` |
| `wang` | `frontend/src/views/admin/`、`frontend/src/views/message/`、`frontend/src/views/group/`、`frontend/src/components/admin/` |

---

## 5. Git 协作规则

### 5.1 不直接改 main

除 `z3` 处理紧急修复外，其他成员不直接向 `main` push。推荐流程：

```bash
git checkout main
git pull origin main
git checkout -b feature/成员代号-功能名
```

示例：

```bash
git checkout -b feature/grand-blog-api
git checkout -b feature/dong-public-pages
git checkout -b feature/dl-editor-ui
git checkout -b feature/wang-message-ui
```

### 5.2 PR 标题格式

```text
[成员][模块] 简短说明
```

示例：

```text
[grand][blog] 实现文章编辑接口
[dong][home] 优化首页照片墙视觉
[dl][editor] 优化文章编辑器体验
[wang][admin] 增加用户审核红点
[z3][ci] 增加 GitHub Actions 自动部署
```

### 5.3 PR 必须说明

每个 PR 描述里必须写：

```text
## 完成内容
## 修改范围
## 自测结果
## 是否影响数据库 / 上传文件 / 权限
## 截图或接口返回示例
```

### 5.4 合并前检查

后端改动至少运行：

```bash
go test ./...
go vet ./...
go build ./cmd/flyteam-server
```

前端静态 JS 改动至少运行：

```bash
node --check app/static/js/你改的文件.js
```

同时 GitHub Actions 必须通过。

---

## 6. 阶段计划

### 第一阶段：稳定当前代码结构和 CI/CD

负责人：`z3`

目标：

- 后端目录继续保持清晰分层。
- GitHub Actions 自动测试构建。
- main 分支部署流程可用。
- 明确所有成员目录所有权。

### 第二阶段：后端能力补齐

负责人：`z3`、`grand`

目标：

- grand 继续完善用户、博客、社交、聊天、RAG API。
- z3 负责数据库、权限、安全、上传、部署和审核。
- 所有 API 要有清晰的请求/响应格式。

### 第三阶段：前端体验升级

负责人：`dong`、`dl`、`wang`

目标：

- dong 完成公开官网视觉统一。
- dl 完成博客和用户中心体验。
- wang 完成后台、聊天、通知体验。
- 三名前端统一字体、色彩、卡片风格、移动端断点。

### 第四阶段：联调和上线

负责人：全体，最终由 `z3` 合并和部署。

目标：

- 全站功能回归测试。
- 权限、安全、上传、XSS、越权检查。
- VPS 数据不丢失。
- CI/CD 自动发布可回滚。

---

## 7. 每个人本周建议任务

| 成员 | 本周优先任务 |
| --- | --- |
| `z3` | 完成 CI/CD、权限安全基线、PR 模板检查、VPS 部署保护 |
| `grand` | 梳理用户/博客/社交 API，补测试，准备 RAG 流式输出方案 |
| `dong` | 统一官网公开页面视觉，重点首页、新闻、回顾、奖项、前辈墙 |
| `dl` | 重做博客广场、文章详情、编辑器、登录注册和个人中心体验 |
| `wang` | 优化后台管理、用户审核提示、私信、群聊和通知页面 |

---

## 8. 冲突处理规则

- 同一个文件不要多人同时改。
- 如果必须多人改同一 CSS 文件，先在群里说明要改哪些 class。
- 后端 API 字段变更必须先通知前端三人。
- 前端需要新 API 时，先在 issue 或 PR 描述里写清楚字段。
- 数据库字段变化必须由 `z3` 确认。
- 安全、鉴权、上传、部署相关修改必须由 `z3` 审核后合并。

---

## 9. 最终验收标准

- GitHub Actions CI 通过。
- 本地 `go test ./...` 通过。
- 首页、官网内容、招新、后台、RAG、博客、聊天全部可用。
- 未登录权限限制正常。
- 管理员和超级管理员权限边界正常。
- 上传文件不能执行脚本，图片/PDF 类型限制正常。
- VPS 部署不覆盖 `.env` 和 `storage/`。
- 五个人的代码都通过 PR 审核后合并。
