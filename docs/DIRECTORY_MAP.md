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
| `cmd/flyteam-server/` | Go 后端命令入口与 internal 分层代码。 |
| `cmd/flyteam-server/internal/app/` | HTTP 应用层：配置、路由、鉴权适配、官网内容、社区接口、RAG 调度与上传处理。 |
| `cmd/flyteam-server/internal/blog/` | 博客领域层：文章模型、发布请求校验、标签规范化、公开响应结构。 |
| `cmd/flyteam-server/internal/database/` | 数据库层：SQLite 连接、Schema 初始化、app_kv JSON 存取。 |
| `docs/` | 项目文档总目录。 |
| `docs/knowledge/` | 本地知识库 PDF 草稿占位目录，PDF 默认不提交 Git。 |
| `docs/planning/` | 规划、路线图、多人成员协作任务分配。 |
| `docs/reports/` | 测试、安全、验收报告。 |
| `scripts/` | 项目维护脚本。 |
| `storage/` | 运行数据目录：数据库、上传文件、RAG 索引和日志；公开协作时默认不建议提交。 |

## 关键文件

| 路径 | 说明 |
| --- | --- |
| `.env.example` | 环境变量模板。 |
| `.github/workflows/ci-cd.yml` | GitHub Actions CI/CD：测试、构建、打包和可选 VPS 自动部署。 |
| `.gitignore` | Git 忽略规则，排除密钥、运行数据、上传文件、日志和本地缓存。 |
| `CONTRIBUTING.md` | 协作流程说明。 |
| `README.md` | 项目总说明和本地运行指南。 |
| `docs/CI_CD.md` | GitHub Actions CI/CD 与 VPS 自动部署配置说明。 |
| `docs/DIRECTORY_MAP.md` | 自动生成的项目目录明细。 |
| `docs/PROJECT_STRUCTURE.md` | 目录结构约定和新增文件放置规范。 |
| `docs/REFACTOR_ARCHITECTURE.md` | 下一阶段 Go + Gin / Vue 前后端分离目标架构说明。 |
| `docs/REFACTOR_REQUIREMENTS.md` | 下一阶段重构与新增功能需求说明。 |
| `docs/REFACTOR_TASK_PLAN.md` | 下一阶段重构任务拆分、阶段计划与成员映射说明。 |
| `docs/planning/blog-community-roadmap.md` | 博客社区化改造路线图。 |
| `docs/planning/team-task-allocation.md` | 五人协作任务分配：z3/grand 后端，dong/dl/wang 前端。 |
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
| `cmd/flyteam-server/internal/app/admin_auth.go` | 管理员后台鉴权、会话、角色权限和管理员账号管理。 |
| `cmd/flyteam-server/internal/app/admin_blog_site_state.go` | 管理员后台博客站开放/关闭状态与访问控制。 |
| `cmd/flyteam-server/internal/app/admin_community_audit.go` | 管理员/超级管理员的社区用户审核、权限管理和聊天审计接口。 |
| `cmd/flyteam-server/internal/app/http_core.go` | HTTP 请求入口、安全响应头和全局前置校验。 |
| `cmd/flyteam-server/internal/app/http_helpers.go` | HTTP/JSON、随机值、限流、时间、路径等通用辅助函数。 |
| `cmd/flyteam-server/internal/app/http_static.go` | 静态资源、上传资源和页面文件安全访问。 |
| `cmd/flyteam-server/internal/app/public_content.go` | 官网前台内容聚合、排序、奖项/前辈墙/新闻等核心逻辑。 |
| `cmd/flyteam-server/internal/app/public_recruit_captcha.go` | 官网前台招新报名动态 C 语言验证码。 |
| `cmd/flyteam-server/internal/app/public_review_recruit.go` | 官网前台团队回顾、相册、招新报名数据处理。 |
| `cmd/flyteam-server/internal/app/routes.go` | 顶层路由分发入口，按静态资源、公共前台、用户前台、管理员后台和 API 分组。 |
| `cmd/flyteam-server/internal/app/routes_admin_backend.go` | 管理员后台页面、管理员 API、后台鉴权/CSRF 权限边界。 |
| `cmd/flyteam-server/internal/app/routes_public_api.go` | 匿名可访问的官网前台只读 API。 |
| `cmd/flyteam-server/internal/app/routes_public_frontend.go` | 官网宣传站公共前台页面路由。 |
| `cmd/flyteam-server/internal/app/routes_recruit.go` | 招新报名公开提交与管理员审核路由。 |
| `cmd/flyteam-server/internal/app/routes_site_admin_content.go` | 宣传站内容管理后台 API 路由。 |
| `cmd/flyteam-server/internal/app/routes_static.go` | 静态文件和上传文件路由。 |
| `cmd/flyteam-server/internal/app/routes_system_api.go` | RAG、文件上传和系统工具 API 路由。 |
| `cmd/flyteam-server/internal/app/routes_user_api.go` | 用户前台博客/社交/私信/群聊 API 路由。 |
| `cmd/flyteam-server/internal/app/routes_user_frontend.go` | 用户前台博客、个人中心、私信、群聊页面路由。 |
| `cmd/flyteam-server/internal/app/server.go` | 服务启动、配置加载和运行时依赖初始化。 |
| `cmd/flyteam-server/internal/app/system_cache.go` | 数据库缓存控制与辅助逻辑。 |
| `cmd/flyteam-server/internal/app/system_database_adapter.go` | 数据库访问适配器，调用 internal/database。 |
| `cmd/flyteam-server/internal/app/system_rag.go` | RAG 知识库、PDF 文本提取、向量检索、问答调用。 |
| `cmd/flyteam-server/internal/app/system_upload.go` | PDF、图片、头像等上传处理和文件安全校验。 |
| `cmd/flyteam-server/internal/app/user_account.go` | 用户前台注册、登录、资料与账号管理。 |
| `cmd/flyteam-server/internal/app/user_blog_articles.go` | 用户前台博客文章发布、编辑、读取、浏览量等。 |
| `cmd/flyteam-server/internal/app/user_blog_interactions.go` | 用户前台博客评论、点赞、收藏等互动。 |
| `cmd/flyteam-server/internal/app/user_blog_model.go` | 用户前台博客领域适配器，调用 internal/blog。 |
| `cmd/flyteam-server/internal/app/user_community_status.go` | 用户前台社区预留/状态接口。 |
| `cmd/flyteam-server/internal/app/user_friends.go` | 用户前台好友申请与好友关系。 |
| `cmd/flyteam-server/internal/app/user_groups.go` | 用户前台群聊、群成员、群管理。 |
| `cmd/flyteam-server/internal/app/user_search_notifications.go` | 用户前台通知与搜索。 |
| `cmd/flyteam-server/internal/app/user_session.go` | 用户前台会话校验、登录态、CSRF 和用户权限辅助。 |
| `cmd/flyteam-server/internal/app/user_social_messages.go` | 用户前台关注、好友、私信等社交消息。 |
| `cmd/flyteam-server/internal/blog/blog.go` | 博客领域模型、文章请求校验、标签规范化和响应转换。 |
| `cmd/flyteam-server/internal/database/database.go` | SQLite 连接、Schema 初始化、app_kv JSON 存取。 |
| `cmd/flyteam-server/main.go` | Go 命令入口，仅调用 internal/app.Run。 |

## 运行时/本地文件说明

| 路径 | 说明 |
| --- | --- |
| `.env` | 本地/服务器真实环境变量，包含密钥，禁止提交到公开仓库。 |
| `storage/flyteam.db` | SQLite 运行数据库，保存账号、内容、文章、聊天等数据。 |
| `storage/uploads/` | 后台上传图片、头像、PDF、博客图片等缓存。 |
| `storage/chroma/` | 旧版 Chroma 向量库缓存，如存在则不建议提交到公开仓库。 |
| `storage/*.json` | 兼容旧版 JSON 数据和迁移来源，不建议提交到公开仓库。 |
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
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.md
│   │   └── feature_request.md
│   └── workflows/
│       └── ci-cd.yml
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
│       ├── main.go
│       └── internal/
│           ├── app/
│           │   ├── admin_auth.go
│           │   ├── admin_blog_site_state.go
│           │   ├── admin_community_audit.go
│           │   ├── admin_community_audit_test.go
│           │   ├── http_core.go
│           │   ├── http_helpers.go
│           │   ├── http_static.go
│           │   ├── public_content.go
│           │   ├── public_recruit_captcha.go
│           │   ├── public_review_recruit.go
│           │   ├── public_senior_sort_test.go
│           │   ├── routes.go
│           │   ├── routes_admin_backend.go
│           │   ├── routes_public_api.go
│           │   ├── routes_public_frontend.go
│           │   ├── routes_recruit.go
│           │   ├── routes_site_admin_content.go
│           │   ├── routes_static.go
│           │   ├── routes_system_api.go
│           │   ├── routes_user_api.go
│           │   ├── routes_user_frontend.go
│           │   ├── server.go
│           │   ├── system_cache.go
│           │   ├── system_database_adapter.go
│           │   ├── system_rag.go
│           │   ├── system_upload.go
│           │   ├── user_account.go
│           │   ├── user_avatar_upload_test.go
│           │   ├── user_blog_articles.go
│           │   ├── user_blog_interactions.go
│           │   ├── user_blog_model.go
│           │   ├── user_community_status.go
│           │   ├── user_community_test.go
│           │   ├── user_friends.go
│           │   ├── user_frontend_auth_test.go
│           │   ├── user_groups.go
│           │   ├── user_search_notifications.go
│           │   ├── user_session.go
│           │   └── user_social_messages.go
│           ├── blog/
│           │   └── blog.go
│           └── database/
│               └── database.go
├── docs/
│   ├── CI_CD.md
│   ├── DIRECTORY_MAP.md
│   ├── PROJECT_STRUCTURE.md
│   ├── REFACTOR_ARCHITECTURE.md
│   ├── REFACTOR_REQUIREMENTS.md
│   ├── REFACTOR_TASK_PLAN.md
│   ├── knowledge/
│   │   └── .gitkeep
│   ├── planning/
│   │   ├── blog-community-roadmap.md
│   │   └── team-task-allocation.md
│   └── reports/
│       └── final-test-security-report.md
├── scripts/
│   └── update_directory_map.py
└── storage/
    ├── admin_users.json
    ├── flyteam.db
    ├── ingest_index.json
    ├── recruit_applications.json
    ├── team_content.json
    ├── chroma/
    │   ├── chroma.sqlite3
    │   ├── 12f74cfc-1261-49d5-a39a-149941f21e29/
    │   │   ├── data_level0.bin
    │   │   ├── header.bin
    │   │   ├── length.bin
    │   │   └── link_lists.bin
    │   └── a639fac9-1731-42ac-adeb-8a5ec16e5c30/
    │       ├── data_level0.bin
    │       ├── header.bin
    │       ├── length.bin
    │       └── link_lists.bin
    └── uploads/
        ├── Flyteam.pdf
        ├── flyteam_knowledge.pdf
        ├── Flyteam团队详情.pdf
        ├── awards/
        │   ├── 02483ac4f6074cb7bc7ecf4cbe935257.jpg
        │   ├── 0d4aa531ffdc43aaa3b3ea712ed24959.png
        │   ├── 17c6ce941f1b46bd8deacfc5cdb07abf.jpg
        │   ├── 29835a731e6248ff804473a797ccacf9.jpg
        │   ├── 30f13a2e64_灞忓箷鎴浘_2026-05-09_010816.png
        │   ├── 38b89118c74f4b86953fd2f6d6069859.jpg
        │   ├── 3ce21d7fefcd4bac8c9e8eb73734e592.jpg
        │   ├── 43393fd00a1e4ef2a9147cd6eab82584.jpg
        │   ├── 44dfd15f95fe435bab9bf8870af3ea25.jpg
        │   ├── 5f65f5a1754546ebb92876010c91d3e0.png
        │   ├── 6002d07ee1ec4643aee47cf78719c607.jpg
        │   ├── 6a5bcec8fdcc45a0b395fa9b6e05fa0a.jpg
        │   ├── 6c3efd46a3d341b7a34734429d19e647.jpg
        │   ├── 8f9760c4c5_58affa29829b601cc7e4849ff73c739d.png
        │   ├── 93c946581ff14bccb9564d70251d81f8.png
        │   ├── 9de412a85b_微信图片_20251224182751_84_130.png
        │   ├── a160223a093c4c57bbfe17aade508653.jpg
        │   ├── a22ead2e82944470a9e64394851e465e.jpg
        │   ├── a9f8e8297f3740488b577a56b7a29193.jpg
        │   ├── aa333b8b206d412fbe9a0851a5790ee9.jpg
        │   ├── ada79b4da6_Snipaste_2026-05-09_00-59-11.jpg
        │   ├── b15c371956_Snipaste_2026-05-09_01-07-23.jpg
        │   ├── b67f3faf07_Snipaste_2026-05-09_01-05-44.jpg
        │   ├── c132d46b20e34ce48b9996f77c1b9287.jpg
        │   ├── d1b2043998284056bb8d362c7298fae4.jpg
        │   ├── d1fd8f0b91_Snipaste_2026-05-09_00-44-01.jpg
        │   ├── e5d1fd183312470a877ef5a9a3723f4d.jpg
        │   ├── e77a90eb96784e388b7adf43ff15b033.jpg
        │   └── ff43e9c2bf6e48a4bc9550cd8e2ed896.jpg
        ├── images/
        │   ├── 05126f1778_寰俊鍥剧墖_2025-08-28_050927_608.jpg
        │   ├── 0667e2ed06_117.jpg
        │   ├── 12d20723f8_124.jpg
        │   ├── 132046b7a0_寰俊鍥剧墖_2025-08-28_051006_106.jpg
        │   ├── 1367f6db44_127.jpg
        │   ├── 1b43af4fa3_114.jpg
        │   ├── 2b894d2122_19d5147c2570875d498a6bcefb76d0cf.jpg
        │   ├── 2c8a4f9c4a_61f92fe1cfff8983ad1326dd5a441cd2.jpg
        │   ├── 33c9d87295_寰俊鍥剧墖_2025-08-28_051030_288.jpg
        │   ├── 375a5e90ff_4.jpg
        │   ├── 3a678c0c81_寰俊鍥剧墖_2025-08-28_050657_196.jpg
        │   ├── 3c9a758006_Image_1758443519893.jpg
        │   ├── 46ecd1d846_寰俊鍥剧墖_2025-08-28_050443_008.jpg
        │   ├── 5cc61bbcef_寰俊鍥剧墖_2025-08-28_050807_223.jpg
        │   ├── 5f881c79d1_226.jpg
        │   ├── 603295cd28_寰俊鍥剧墖_2025-08-28_050641_674.jpg
        │   ├── 65424b99d4_寰俊鍥剧墖_2025-08-28_050745_790.jpg
        │   ├── 708c2c87ad_6.jpg
        │   ├── 7c06fb9ad9_寰俊鍥剧墖_2025-08-28_051026_518.jpg
        │   ├── 81fc178491_126.jpg
        │   ├── 823caf3392_Image_1758443506728.jpg
        │   ├── 83195570b6_寰俊鍥剧墖_2025-08-28_050834_547.jpg
        │   ├── 8337e21ffd_5.jpg
        │   ├── 84386816f8_a07be91b2a41d13bb4c064c6283a86a3.jpg
        │   ├── 8579721af7_2.jpg
        │   ├── 8935c4f5ed_寰俊鍥剧墖_2025-08-28_050714_725.jpg
        │   ├── 9488215d0b_113.jpg
        │   ├── 95f9dc94ef_742690305b45eadc194dd5a18e0b2cd5.jpg
        │   ├── 9c29a97d22_寰俊鍥剧墖_2025-08-28_050515_307_1_1.png
        │   ├── a34359c6f8_寰俊鍥剧墖_2025-08-28_050734_315.jpg
        │   ├── acda61215e_123.jpg
        │   ├── adf310ab52_115.jpg
        │   ├── bee3cbe3f9_121.jpg
        │   ├── c3fa211efc_728874b22854db485366f7673da9345e.jpg
        │   ├── c5094685dd_128.jpg
        │   ├── c8c5a02fd1_110.jpg
        │   ├── de28770e77_122.jpg
        │   ├── e040edb3fe_111.jpg
        │   ├── f1d320943f_118.jpg
        │   ├── f1f8e18660_224.jpg
        │   ├── f72f6b5475_3.jpg
        │   └── f93749f93d_寰俊鍥剧墖_2025-08-28_050751_435.jpg
        ├── news/
        │   ├── 07bad01e19.jpg
        │   ├── 0a7a61f589_寰俊鍥剧墖_2026-04-26_215953_590.jpg
        │   ├── 0a7a61f589_微信图片_2026-04-26_215953_590.jpg
        │   ├── 0ef80de3b61c466dba94b9545536453c.jpg
        │   ├── 0f41b56e86c449f08c945b0b52bfadea.jpg
        │   ├── 1033a29d94.png
        │   ├── 111c328acd044711888df2100a267a69.jpg
        │   ├── 131ef9fe33.jpg
        │   ├── 186ec9e09233425492ef633868af9951.jpg
        │   ├── 1cebcf8d2d7d4c6b8fc95e17c744a870.jpg
        │   ├── 1cf8c350bd614443a1fd8f9cc00a8de6.jpg
        │   ├── 22ab1b5341.png
        │   ├── 2665109615.jpg
        │   ├── 28b318680a.png
        │   ├── 2c742fa640.jpg
        │   ├── 2f9610ef08ae451eb956dc95d0a9587f.png
        │   ├── 30792b1c9dac46ba97e82bd543c6b718.jpg
        │   ├── 33988d72a2cc4fb982b299f551ef8041.jpg
        │   ├── 347da35697.jpg
        │   ├── 36dc51e20e.png
        │   ├── 3baf09df932a46d9ab94f7d3f5d5899b.jpg
        │   ├── 3edc04582e.png
        │   ├── 40a4062e7b.jpg
        │   ├── 441e5f6e6f.png
        │   ├── 4a00a04fcb.jpg
        │   ├── 4c80304b4f_寰俊鍥剧墖_2026-04-26_220152_809.png
        │   ├── 4c80304b4f_微信图片_2026-04-26_220152_809.png
        │   ├── 4e46bc24ac.jpg
        │   ├── 6303e738ed_728874b22854db485366f7673da9345e.jpg
        │   ├── 63b4692983.jpg
        │   ├── 66a67f40a021422ba308bd8132234468.jpg
        │   ├── 67c43f5b853446828a7455ac4f78691d.png
        │   ├── 67d0a65d5d.jpg
        │   ├── 6ab2e67c6b.jpg
        │   ├── 735dc960467d48608fe4b5ec0be6ecc4.jpg
        │   ├── 7742926db9.png
        │   ├── 7b719e515c.jpg
        │   ├── 7b76700bce4746a58c6733b6d6055c7e.jpg
        │   ├── 7d073a0887.jpg
        │   ├── 7d0e3126f4.jpg
        │   ├── 7e03a180db.jpg
        │   ├── 833c5fb3ae.jpg
        │   ├── 8cd23437e7.jpg
        │   ├── 9291395bda.jpg
        │   ├── 95434ff755.jpg
        │   ├── 95a2160a00cd44beafd43ebf6342ea80.jpg
        │   ├── 9639904694.png
        │   ├── a0d260759e.png
        │   ├── a3aa3454c1784f1993633a2ac1555fed.jpg
        │   ├── a6f716e64f0c4c2ea54bca73fbade9a0.jpg
        │   ├── a7049fd9a8.jpg
        │   ├── a8148a1046.jpg
        │   ├── a909e89844.png
        │   ├── ac3b0c4775.jpg
        │   ├── ae3b53e82b.jpg
        │   ├── b7f69d9c18.png
        │   ├── b9904e41d4724bb49ac8b708c6b87b89.jpg
        │   ├── bd85ccf60b_寰俊鍥剧墖_2026-04-26_220232_368.png
        │   ├── bd85ccf60b_微信图片_2026-04-26_220232_368.png
        │   ├── c0e874cd61494e5baa29801b157423a9.jpg
        │   ├── c996eb4c92.png
        │   ├── cb79268bc16648b9ba46f9eb13234933.jpg
        │   ├── cdf25668e0.png
        │   ├── d6420c9c23.jpg
        │   ├── db043e5e3f.png
        │   ├── e08d74634d.jpg
        │   ├── e2ede8fc9f.png
        │   ├── e370559230.jpg
        │   ├── e59603590f.jpg
        │   ├── ef8056021e.jpg
        │   ├── fd35255821.jpg
        │   ├── fd50f42c58.jpg
        │   └── ff7e20080f2b44f4862e2268e83394c3.jpg
        ├── review/
        │   ├── 23e378b82fd649269584708221323e04.jpg
        │   ├── 3eb2aae8e540454895fca3eceb85d8b2.jpg
        │   ├── 6502b3276e.jpg
        │   ├── 91472a9d1d744bfa8cabeb6377e8e286.jpg
        │   ├── 972d964cb10a491f8d36be832da58c5d.jpg
        │   ├── a4a16fb26d.jpg
        │   ├── f6ab6075a50c4a099d017bf291098769.jpg
        │   └── f73de2daa8e04382ac38d4f6f826d4ad.png
        └── seniors/
            ├── 0cf3c098e6_1NR4dwtRbEQ0xCuPgfkHbZ_BX8FwVo3nvT5Q7Gz926A.png
            ├── 0d5f933355cd4149b3172e94560d1e4a.png
            ├── 0ffebe5334bc4656875efae1f68ba902.jpg
            ├── 13cdac08288e4c3fb38e79ee48ff6bb8.jpg
            ├── 19c153e939.png
            ├── 23a0d2e9095c4105a2c8ffd9409d2d4e.jpg
            ├── 26a6936d751f4d45ba765d67ab7526a1.jpg
            ├── 352230e750d84301b264a073a3ad0d4c.jpg
            ├── 41dd7362ec.png
            ├── 504228e5b54e4f8da36f16185a15fc37.jpg
            ├── 6a5363183d8842528b7948026f4e1c97.png
            ├── 6fcd0f4650b646e5b2a51290d16d89ea.png
            ├── 81c1c675c90b4a94a6227a7952539f24.png
            ├── 946b5f40c18c45a9bde53d31e1311c5a.png
            ├── 9b5b22142f_Ju18CTU7uGuavBVqXottCV8EU3SBrAQpTH2GRxryQ2Q.jpg
            ├── a67f0daf7c.jpg
            ├── a7ab749ef1_微信图片_2026-04-26_002548_855.png
            ├── bb2f0ccb6e3f455f8e7844b751ae102f.jpg
            ├── c0a965b74de4408cbfc74507f38ed348.jpg
            ├── e2e401462b024227ad06159f6e1683ef.png
            ├── e314a2c34de348629841913f33600aad.png
            ├── f1aab35c90f84c6782153a7b286a292b.jpg
            ├── f76224a2ee.jpg
            ├── fd853312acfa41e2a27a56922e12c78f.jpg
            ├── fdf82e3105e74236bb1b5a3658a7b978.jpg
            └── fe42071984_e074d5bf71d0835ffb1f5b9bcf8b121a.jpg
```
