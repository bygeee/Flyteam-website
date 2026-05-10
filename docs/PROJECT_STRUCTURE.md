# 项目目录结构说明

本项目已经按「代码 / 前端资源 / 文档 / 运行数据」重新归类，后续协作请尽量遵守以下结构。

```text
.
├── cmd/flyteam-server/        # Go 后端入口和业务模块
├── app/static/
│   ├── pages/                 # HTML 页面模板，由 Go 路由加载
│   ├── js/                    # 前端 JS
│   ├── css/                   # 前端样式
│   └── assets/                # 预留静态图片、字体等公共资源
├── docs/
│   ├── planning/              # 规划、任务拆分、路线图
│   ├── reports/               # 测试、安全、验收报告
│   └── knowledge/             # 本地知识库 PDF 草稿/源文件，默认不提交
├── storage/                   # 运行数据、数据库、上传文件，默认不提交
├── archive/legacy-python/     # 旧 Python 版本备份，默认不提交
├── .github/                   # PR / Issue / CODEOWNERS 配置
├── README.md                  # 项目说明和本地运行指南
├── CONTRIBUTING.md            # 协作流程
├── go.mod / go.sum            # Go 依赖
└── .env.example               # 环境变量模板
```

## 约定

- 页面新增到 `app/static/pages/`。
- JS 新增到 `app/static/js/`，HTML 中通过 `/static/js/xxx.js` 引用。
- CSS 新增到 `app/static/css/`，HTML 中通过 `/static/css/xxx.css` 引用。
- 图片、字体等公共静态资源放到 `app/static/assets/`。
- 用户上传内容、数据库、RAG 缓存继续放在 `storage/`，不要提交到 Git。
- 大型 PDF 或本地知识库源文件放在 `docs/knowledge/`，默认仍受 `*.pdf` 忽略规则保护。

## 后端路由说明

Go 服务仍以 `cmd/flyteam-server` 为入口。`serveStaticHTML` 会从 `app/static/pages/` 读取页面；`/static/js/` 和 `/static/css/` 由静态文件服务提供。

为了兼容旧链接，`/static/*.html` 仍会映射到 `app/static/pages/*.html`；但新的页面开发不要再使用旧路径。
