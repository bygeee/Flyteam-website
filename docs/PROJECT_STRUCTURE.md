# 项目目录结构说明

本项目已经按「代码 / 前端资源 / 文档 / 运行数据」重新归类。后端也已经从单目录文件堆叠调整为 Go `internal` 分层，后续协作请尽量遵守以下结构。

```text
.
├── cmd/flyteam-server/        # Go 后端命令入口
│   ├── main.go                # 极简启动入口，只调用 internal/app.Run
│   └── internal/
│       ├── app/               # HTTP 应用层，继续按 admin_/user_/public_/routes_/system_/http_ 前缀拆分
│       ├── blog/              # 博客领域层：文章模型、发布校验、标签规范化、公开响应结构
│       └── database/          # 数据库层：SQLite 连接、Schema 初始化、app_kv JSON 存取
├── app/static/
│   ├── pages/                 # HTML 页面模板，由 Go 路由加载
│   ├── js/                    # 前端 JS
│   ├── css/                   # 前端样式
│   └── assets/                # 预留静态图片、字体等公共资源
├── docs/
│   ├── planning/              # 规划、任务拆分、路线图
│   ├── reports/               # 测试、安全、验收报告
│   └── knowledge/             # 本地知识库 PDF 草稿/源文件，默认不提交
├── storage/                   # 运行数据、数据库、上传文件，公开协作时默认不建议提交
├── archive/legacy-python/     # 旧 Python 版本备份，默认不提交
├── .github/                   # PR / Issue / CODEOWNERS 配置
├── README.md                  # 项目说明和本地运行指南
├── CONTRIBUTING.md            # 协作流程
├── go.mod / go.sum            # Go 依赖
└── .env.example               # 环境变量模板
```

## 放置约定

- 页面新增到 `app/static/pages/`。
- JS 新增到 `app/static/js/`，HTML 中通过 `/static/js/xxx.js` 引用。
- CSS 新增到 `app/static/css/`，HTML 中通过 `/static/css/xxx.css` 引用。
- 图片、字体等公共静态资源放到 `app/static/assets/`。
- 用户上传内容、数据库、RAG 缓存继续放在 `storage/`；公开协作时不要提交真实运行数据。
- 大型 PDF 或本地知识库源文件放在 `docs/knowledge/`，默认仍受 `*.pdf` 忽略规则保护。

## 后端分层说明

Go 服务仍以 `cmd/flyteam-server/main.go` 为入口，主业务位于 `cmd/flyteam-server/internal/app/`。

- `internal/app/`：负责 HTTP Server 生命周期、路由注册、请求鉴权、静态页面渲染、官网内容管理、用户社区 API、RAG 调度、上传入口等应用编排逻辑。该目录内继续按文件名前缀拆分：
  - `admin_*.go`：管理员后台、超级管理员、用户审核、后台权限、后台审计。
  - `user_*.go`：普通用户前台、博客、个人中心、好友、私信、群聊、通知搜索。
  - `public_*.go`：官网宣传前台、团队新闻、团队回顾、奖项荣誉、前辈墙、招新报名。
  - `routes_*.go`：路由分组，分别承接用户前台、管理员后台、公共前台、招新、上传/RAG 等入口。
  - `system_*.go`：缓存、数据库适配、RAG、上传等系统基础设施。
  - `http_*.go`：HTTP 入口、安全响应头、静态文件、JSON/限流等公共能力。
- `internal/database/`：负责 SQLite 打开、连接参数、基础 Schema 初始化和通用 JSON KV 存储。后续新增数据库迁移、索引、备份恢复等都优先放这里。
- `internal/blog/`：负责博客文章领域模型、发布/编辑请求校验、标签规范化和对外响应结构。后续博客推荐、文章草稿规则、文章审核规则等优先放这里。

为了兼容当前已经完成的路由和测试，`internal/app` 中保留少量适配文件，例如 `system_database_adapter.go` 调用 `internal/database`，`user_blog_model.go` 调用 `internal/blog`。后续新增模块也按这个方式拆：纯业务规则先进独立领域包，HTTP handler 只做请求解析、权限校验和调用编排。

## 前端路由说明

`serveStaticHTML` 会从 `app/static/pages/` 读取页面；`/static/js/` 和 `/static/css/` 由静态文件服务提供。

为了兼容旧链接，`/static/*.html` 仍会映射到 `app/static/pages/*.html`；但新的页面开发不要再使用旧路径。

## 目录明细更新

`docs/DIRECTORY_MAP.md` 是自动生成的当前目录明细。每次新增、移动、删除目录或关键文件后，请运行：

```bash
python scripts/update_directory_map.py
```

并把更新后的 `docs/DIRECTORY_MAP.md` 一起提交。
