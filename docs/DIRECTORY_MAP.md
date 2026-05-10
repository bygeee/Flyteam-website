# 项目目录明细

> 本文件由 `scripts/update_directory_map.py` 自动生成。每次新增、移动、删除目录或关键文件后，请重新运行脚本并提交本文件。

## 更新方式

```bash
python scripts/update_directory_map.py
```

Windows PowerShell：

```powershell
python scripts/update_directory_map.py
```

## 顶层目录与职责

| 路径 | 说明 |
| --- | --- |
| `.github/` | GitHub 协作配置：CODEOWNERS、Issue 模板、PR 模板。 |
| `app/static/assets/` | 公共静态资源预留目录：图片、字体、默认背景等。 |
| `app/static/css/` | 前端样式文件。 |
| `app/static/js/` | 前端交互脚本。 |
| `app/static/pages/` | HTML 页面模板，由 Go 后端路由加载。 |
| `archive/legacy-python/` | 旧 Python 版本备份占位目录，真实备份文件默认不提交。 |
| `cmd/flyteam-server/` | Go 后端服务，包含路由、鉴权、内容管理、博客社区、RAG、上传等模块。 |
| `docs/` | 项目文档总目录。 |
| `docs/knowledge/` | 本地知识库/PDF 草稿占位目录，PDF 默认不提交 Git。 |
| `docs/planning/` | 规划、路线图、多人协作任务分配。 |
| `docs/reports/` | 测试、安全、验收报告。 |
| `scripts/` | 项目维护脚本。 |
| `storage/` | 运行数据目录：数据库、上传文件、RAG 索引和日志，默认不提交。 |

## 关键文件

| 路径 | 说明 |
| --- | --- |
| `.env.example` | 环境变量模板。 |
| `.gitignore` | Git 忽略规则，排除密钥、运行数据、上传文件、日志和本地缓存。 |
| `CONTRIBUTING.md` | 协作流程说明。 |
| `README.md` | 项目总说明和本地运行指南。 |
| `docs/DIRECTORY_MAP.md` | 自动生成的项目目录明细。 |
| `docs/PROJECT_STRUCTURE.md` | 目录结构约定和新增文件放置规范。 |
| `docs/planning/blog-community-roadmap.md` | 博客社区化改造路线图。 |
| `docs/planning/team-task-allocation.md` | z3 / grand / dl 任务分配。 |
| `docs/reports/final-test-security-report.md` | 功能与安全测试报告。 |
| `go.mod` | Go 模块定义。 |
| `go.sum` | Go 依赖校验锁定文件。 |
| `scripts/update_directory_map.py` | 自动刷新 docs/DIRECTORY_MAP.md。 |

## 前端资源明细

| 目录 | 类型 | 当前文件 |
| --- | --- | --- |
| `app/static/pages/` | 页面模板 | account.html, admin.html, article.html, awards.html, blog.html, editor.html, flyteamers.html, groups.html, index.html, intro.html, login.html, messages.html, news.html, recruit.html, review.html, review_detail.html, space.html, user_login.html, user_register.html |
| `app/static/js/` | 前端脚本 | account.js, app.js, article.js, blog.js, editor.js, flyteamers.js, groups.js, login.js, messages.js, news.js, page_backdrop.js, public.js, recruit.js, space.js, user_login.js, user_register.js |
| `app/static/css/` | 样式文件 | community.css, public.css, styles.css |

## Go 后端模块明细

| 路径 | 说明 |
| --- | --- |
| `cmd/flyteam-server/admin_blog_ops.go` | 博客站管理员/超级管理员操作、用户审核、审计接口。 |
| `cmd/flyteam-server/auth.go` | 宣传站管理员鉴权、会话、角色权限。 |
| `cmd/flyteam-server/blog_site_state.go` | 博客站开放/关闭状态与前端访问控制。 |
| `cmd/flyteam-server/cache.go` | 缓存控制与辅助逻辑。 |
| `cmd/flyteam-server/captcha.go` | 招新报名动态 C 语言验证码。 |
| `cmd/flyteam-server/community_auth.go` | 社区鉴权公共逻辑。 |
| `cmd/flyteam-server/community_blog.go` | 博客文章发布、编辑、读取、浏览量等。 |
| `cmd/flyteam-server/community_dl_comments.go` | 博客评论、点赞、收藏等互动。 |
| `cmd/flyteam-server/community_dl_groups.go` | 群聊、群成员、群管理。 |
| `cmd/flyteam-server/community_dl_notify_search.go` | 通知与搜索。 |
| `cmd/flyteam-server/community_dl_routes.go` | 社区 API 路由分发。 |
| `cmd/flyteam-server/community_dl_social_messages.go` | 关注、好友、私信等社交消息。 |
| `cmd/flyteam-server/community_friends.go` | 好友申请与好友关系。 |
| `cmd/flyteam-server/community_grand_auth.go` | 社区用户注册、登录、资料与账号管理。 |
| `cmd/flyteam-server/community_reserved.go` | 社区预留/状态接口。 |
| `cmd/flyteam-server/content.go` | 官网内容聚合、排序、奖项/前辈墙/新闻等核心内容逻辑。 |
| `cmd/flyteam-server/content_review_recruit.go` | 团队回顾、相册、招新报名数据处理。 |
| `cmd/flyteam-server/database.go` | SQLite 初始化、表结构迁移、默认账号/数据迁移。 |
| `cmd/flyteam-server/main.go` | 服务入口、配置加载、HTTP 路由、静态文件服务和安全响应头。 |
| `cmd/flyteam-server/rag.go` | RAG 知识库、PDF 文本提取、向量检索、问答调用。 |
| `cmd/flyteam-server/upload.go` | PDF、图片、头像等上传处理和文件安全校验。 |

## 运行时/本地文件说明

| 路径 | 说明 |
| --- | --- |
| `.env` | 本地/服务器真实环境变量，包含密钥，禁止提交。 |
| `storage/flyteam.db` | SQLite 运行数据库，保存账号、内容、文章、聊天等数据。 |
| `storage/uploads/` | 后台上传图片、头像、PDF、博客图片等缓存。 |
| `storage/chroma/` | 旧版 Chroma 向量库缓存，如存在则不提交。 |
| `storage/*.json` | 兼容旧版 JSON 数据和迁移来源，不提交。 |
| `storage/*.log` | 运行日志，不提交。 |
| `.venv/` | 本地 Python 虚拟环境，不提交。 |
| `archive/legacy-python/*.codex_backup` | 旧 Python 备份文件，不提交。 |

## 当前 Git 跟踪文件树

```text
.
├── .env.example
├── .gitignore
├── CONTRIBUTING.md
├── go.mod
├── go.sum
├── README.md
├── .github/
│   ├── CODEOWNERS
│   ├── PULL_REQUEST_TEMPLATE.md
│   └── ISSUE_TEMPLATE/
│       ├── bug_report.md
│       └── feature_request.md
├── app/
│   └── static/
│       ├── assets/
│       │   └── .gitkeep
│       ├── css/
│       │   ├── community.css
│       │   ├── public.css
│       │   └── styles.css
│       ├── js/
│       │   ├── account.js
│       │   ├── app.js
│       │   ├── article.js
│       │   ├── blog.js
│       │   ├── editor.js
│       │   ├── flyteamers.js
│       │   ├── groups.js
│       │   ├── login.js
│       │   ├── messages.js
│       │   ├── news.js
│       │   ├── page_backdrop.js
│       │   ├── public.js
│       │   ├── recruit.js
│       │   ├── space.js
│       │   ├── user_login.js
│       │   └── user_register.js
│       └── pages/
│           ├── account.html
│           ├── admin.html
│           ├── article.html
│           ├── awards.html
│           ├── blog.html
│           ├── editor.html
│           ├── flyteamers.html
│           ├── groups.html
│           ├── index.html
│           ├── intro.html
│           ├── login.html
│           ├── messages.html
│           ├── news.html
│           ├── recruit.html
│           ├── review.html
│           ├── review_detail.html
│           ├── space.html
│           ├── user_login.html
│           └── user_register.html
├── archive/
│   └── legacy-python/
│       └── .gitkeep
├── cmd/
│   └── flyteam-server/
│       ├── admin_blog_ops.go
│       ├── admin_blog_ops_test.go
│       ├── auth.go
│       ├── avatar_upload_test.go
│       ├── blog_site_state.go
│       ├── cache.go
│       ├── captcha.go
│       ├── community_auth.go
│       ├── community_blog.go
│       ├── community_dl_comments.go
│       ├── community_dl_groups.go
│       ├── community_dl_notify_search.go
│       ├── community_dl_routes.go
│       ├── community_dl_social_messages.go
│       ├── community_dl_test.go
│       ├── community_friends.go
│       ├── community_frontend_auth_test.go
│       ├── community_grand_auth.go
│       ├── community_reserved.go
│       ├── content.go
│       ├── content_review_recruit.go
│       ├── content_senior_sort_test.go
│       ├── database.go
│       ├── main.go
│       ├── rag.go
│       └── upload.go
├── docs/
│   ├── DIRECTORY_MAP.md
│   ├── PROJECT_STRUCTURE.md
│   ├── knowledge/
│   │   └── .gitkeep
│   ├── planning/
│   │   ├── blog-community-roadmap.md
│   │   └── team-task-allocation.md
│   └── reports/
│       └── final-test-security-report.md
└── scripts/
    └── update_directory_map.py
```
