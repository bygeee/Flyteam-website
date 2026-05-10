# Flyteam Website 重构后项目代码结构文档

## 1. 目标架构

重构后的项目采用前后端分离结构：

- 后端：Go + Gin
- 前端：Vue 3 + Vite
- 数据库：SQLite，后续可平滑迁移到 MySQL/PostgreSQL
- 部署：Docker Compose
- 静态资源：前端由 Nginx 提供，上传文件由后端管理

核心调用链路：

```text
Vue 页面
  -> frontend/src/api/*
  -> Gin Handler
  -> Service
  -> Repository
  -> SQLite / 文件系统 / RAG 模型服务
```

## 2. 顶层目录结构

```text
Flyteam-website/
├─ backend/                         # Go + Gin 后端
├─ frontend/                        # Vue 3 + Vite 前端
├─ deploy/                          # 部署配置
├─ docs/                            # 项目文档
├─ scripts/                         # 项目维护脚本
├─ storage/                         # 本地运行挂载数据，可不提交
├─ docker-compose.yml
├─ .env.example
├─ .gitignore
├─ README.md
└─ CONTRIBUTING.md
```

## 3. 后端代码结构

```text
backend/
├─ cmd/
│  └─ server/
│     └─ main.go                    # 后端启动入口
│
├─ internal/
│  ├─ api/                          # API 层：Gin 路由和 handler
│  │  ├─ router.go                  # Gin Router 总入口
│  │  └─ v1/
│  │     ├─ health_handler.go
│  │     ├─ auth_handler.go
│  │     ├─ admin_handler.go
│  │     ├─ site_handler.go         # 官网内容
│  │     ├─ recruit_handler.go      # 招新
│  │     ├─ upload_handler.go
│  │     ├─ rag_handler.go          # RAG + SSE 流式输出
│  │     ├─ user_handler.go
│  │     ├─ blog_handler.go
│  │     ├─ social_handler.go
│  │     ├─ message_handler.go
│  │     ├─ group_handler.go
│  │     ├─ notification_handler.go
│  │     ├─ search_handler.go
│  │     ├─ announcement_handler.go
│  │     ├─ tool_handler.go
│  │     ├─ navigation_handler.go
│  │     └─ maintenance_handler.go
│  │
│  ├─ service/                      # 业务逻辑层
│  │  ├─ auth_service.go
│  │  ├─ admin_service.go
│  │  ├─ site_service.go
│  │  ├─ recruit_service.go
│  │  ├─ upload_service.go
│  │  ├─ rag_service.go
│  │  ├─ user_service.go
│  │  ├─ blog_service.go
│  │  ├─ social_service.go
│  │  ├─ message_service.go
│  │  ├─ group_service.go
│  │  ├─ notification_service.go
│  │  ├─ search_service.go
│  │  ├─ announcement_service.go
│  │  ├─ tool_service.go
│  │  ├─ navigation_service.go
│  │  └─ maintenance_service.go
│  │
│  ├─ repository/                   # 数据访问层，只在这里写 SQL
│  │  ├─ db.go
│  │  ├─ transaction.go
│  │  ├─ admin_repository.go
│  │  ├─ site_repository.go
│  │  ├─ recruit_repository.go
│  │  ├─ rag_repository.go
│  │  ├─ user_repository.go
│  │  ├─ blog_repository.go
│  │  ├─ social_repository.go
│  │  ├─ message_repository.go
│  │  ├─ group_repository.go
│  │  ├─ notification_repository.go
│  │  ├─ announcement_repository.go
│  │  ├─ tool_repository.go
│  │  ├─ navigation_repository.go
│  │  ├─ maintenance_repository.go
│  │  └─ audit_repository.go
│  │
│  ├─ model/                        # 数据库实体模型
│  │  ├─ admin.go
│  │  ├─ user.go
│  │  ├─ site.go
│  │  ├─ recruit.go
│  │  ├─ blog.go
│  │  ├─ social.go
│  │  ├─ message.go
│  │  ├─ group.go
│  │  ├─ notification.go
│  │  ├─ rag.go
│  │  ├─ announcement.go
│  │  ├─ tool.go
│  │  ├─ navigation.go
│  │  ├─ maintenance.go
│  │  └─ audit.go
│  │
│  ├─ dto/                          # 请求 / 响应结构
│  │  ├─ common.go
│  │  ├─ auth_dto.go
│  │  ├─ admin_dto.go
│  │  ├─ site_dto.go
│  │  ├─ recruit_dto.go
│  │  ├─ upload_dto.go
│  │  ├─ rag_dto.go
│  │  ├─ user_dto.go
│  │  ├─ blog_dto.go
│  │  ├─ social_dto.go
│  │  ├─ message_dto.go
│  │  ├─ group_dto.go
│  │  ├─ notification_dto.go
│  │  ├─ announcement_dto.go
│  │  ├─ tool_dto.go
│  │  ├─ navigation_dto.go
│  │  └─ maintenance_dto.go
│  │
│  ├─ middleware/                   # Gin 中间件
│  │  ├─ request_id.go
│  │  ├─ logger.go
│  │  ├─ recovery.go
│  │  ├─ cors.go
│  │  ├─ auth.go
│  │  ├─ admin_auth.go
│  │  ├─ csrf.go
│  │  ├─ rate_limit.go
│  │  └─ maintenance.go
│  │
│  ├─ config/
│  │  └─ config.go                  # 环境变量、路径、模型配置
│  │
│  ├─ logger/
│  │  └─ logger.go                  # slog / zap 初始化
│  │
│  ├─ storage/                      # 上传、文件路径、PDF 提取
│  │  ├─ upload.go
│  │  ├─ image.go
│  │  └─ pdf.go
│  │
│  ├─ rag/                          # RAG 领域能力
│  │  ├─ splitter.go
│  │  ├─ embedding.go
│  │  ├─ retriever.go
│  │  ├─ chat.go
│  │  ├─ stream.go                  # SSE / 流式输出
│  │  └─ prompt_guard.go
│  │
│  ├─ security/
│  │  ├─ password.go
│  │  ├─ token.go
│  │  ├─ captcha.go
│  │  └─ sanitize.go
│  │
│  └─ util/
│     ├─ time.go
│     ├─ id.go
│     ├─ pagination.go
│     └─ response.go
│
├─ migrations/                      # 数据库迁移
│  ├─ 001_init.sql
│  ├─ 002_site_content_tables.sql
│  ├─ 003_recruit_table.sql
│  ├─ 004_announcements.sql
│  ├─ 005_tools_navigation.sql
│  ├─ 006_maintenance_audit.sql
│  └─ 007_rag_stream_history.sql
│
├─ scripts/
│  ├─ migrate_json_to_db.go          # 旧 JSON 数据迁移
│  └─ seed_admin.go
│
├─ storage/                         # 容器内挂载目录
│  ├─ uploads/
│  ├─ logs/
│  └─ flyteam.db
│
├─ go.mod
├─ go.sum
└─ Dockerfile
```

## 4. 前端代码结构

```text
frontend/
├─ src/
│  ├─ api/                          # API 请求封装
│  │  ├─ request.ts
│  │  ├─ auth.ts
│  │  ├─ admin.ts
│  │  ├─ site.ts
│  │  ├─ recruit.ts
│  │  ├─ upload.ts
│  │  ├─ rag.ts
│  │  ├─ user.ts
│  │  ├─ blog.ts
│  │  ├─ social.ts
│  │  ├─ message.ts
│  │  ├─ group.ts
│  │  ├─ notification.ts
│  │  ├─ announcement.ts
│  │  ├─ tools.ts
│  │  ├─ navigation.ts
│  │  └─ maintenance.ts
│  │
│  ├─ router/
│  │  └─ index.ts
│  │
│  ├─ stores/                       # Pinia 状态
│  │  ├─ auth.ts
│  │  ├─ admin.ts
│  │  ├─ user.ts
│  │  ├─ site.ts
│  │  └─ maintenance.ts
│  │
│  ├─ layouts/
│  │  ├─ PublicLayout.vue
│  │  ├─ AdminLayout.vue
│  │  ├─ UserLayout.vue
│  │  └─ MaintenanceLayout.vue
│  │
│  ├─ views/
│  │  ├─ public/
│  │  │  ├─ Home.vue
│  │  │  ├─ Intro.vue
│  │  │  ├─ NewsList.vue
│  │  │  ├─ NewsDetail.vue
│  │  │  ├─ Awards.vue
│  │  │  ├─ Review.vue
│  │  │  ├─ ReviewDetail.vue
│  │  │  └─ Flyteamers.vue
│  │  │
│  │  ├─ admin/
│  │  │  ├─ Login.vue
│  │  │  ├─ Dashboard.vue
│  │  │  ├─ SiteContent.vue
│  │  │  ├─ RecruitManage.vue
│  │  │  ├─ UserManage.vue
│  │  │  ├─ BlogAudit.vue
│  │  │  ├─ UploadManage.vue
│  │  │  ├─ AnnouncementManage.vue
│  │  │  ├─ ToolManage.vue
│  │  │  ├─ NavigationManage.vue
│  │  │  ├─ MaintenanceManage.vue
│  │  │  └─ AuditLogs.vue
│  │  │
│  │  ├─ recruit/
│  │  │  └─ RecruitApply.vue
│  │  │
│  │  ├─ blog/
│  │  │  ├─ BlogHome.vue
│  │  │  ├─ ArticleDetail.vue
│  │  │  └─ Editor.vue
│  │  │
│  │  ├─ user/
│  │  │  ├─ UserLogin.vue
│  │  │  ├─ UserRegister.vue
│  │  │  ├─ Account.vue
│  │  │  └─ Space.vue
│  │  │
│  │  ├─ social/
│  │  │  ├─ Messages.vue
│  │  │  ├─ Groups.vue
│  │  │  └─ Notifications.vue
│  │  │
│  │  ├─ rag/
│  │  │  └─ RagChat.vue
│  │  │
│  │  ├─ tools/
│  │  │  └─ Tools.vue
│  │  │
│  │  ├─ navigation/
│  │  │  └─ Navigation.vue
│  │  │
│  │  └─ maintenance/
│  │     └─ Maintenance.vue
│  │
│  ├─ components/
│  ├─ composables/
│  │  ├─ useAuth.ts
│  │  ├─ usePagination.ts
│  │  ├─ useUpload.ts
│  │  └─ useRagStream.ts
│  │
│  ├─ utils/
│  ├─ styles/
│  ├─ App.vue
│  └─ main.ts
│
├─ public/
├─ index.html
├─ package.json
├─ vite.config.ts
└─ Dockerfile
```

## 5. 部署目录结构

```text
deploy/
├─ nginx.conf                       # 前端静态资源 + /api 反向代理
├─ docker-compose.prod.yml
└─ env.example
```

根目录保留：

```text
docker-compose.yml                  # 本地开发和基础部署入口
.env.example                        # 全局环境变量模板
storage/                            # 本地运行数据挂载目录
```

## 6. 文档目录结构

```text
docs/
├─ REFACTOR_REQUIREMENTS.md         # 重构需求文档
├─ REFACTOR_ARCHITECTURE.md         # 架构与代码结构
├─ API_DESIGN.md                    # API 设计
├─ DATABASE_DESIGN.md               # 数据库设计
├─ RAG_STREAMING.md                 # RAG 流式输出设计
├─ DOCKER_DEPLOYMENT.md             # Docker 部署
└─ MIGRATION_PLAN.md                # 旧数据迁移计划
```

## 7. 后端分层职责

### 7.1 API 层

API 层位于 `backend/internal/api`。

职责：

- 注册 Gin 路由
- 解析 path、query、body 参数
- 调用权限中间件
- 调用 service
- 返回统一 JSON 或 SSE 响应

不应包含：

- SQL
- 复杂业务规则
- 文件系统细节
- RAG 检索细节

### 7.2 Service 层

Service 层位于 `backend/internal/service`。

职责：

- 编排业务流程
- 调用 repository
- 调用 storage、rag、security 等基础能力
- 做业务级校验
- 组织事务边界

### 7.3 Repository 层

Repository 层位于 `backend/internal/repository`。

职责：

- 封装所有 SQL
- 管理查询、插入、更新、删除
- 提供事务辅助
- 屏蔽底层数据库细节

约束：

- SQL 只能出现在该层
- 不处理 HTTP 请求
- 不直接返回前端 DTO

### 7.4 Model 层

Model 层位于 `backend/internal/model`。

职责：

- 定义数据库实体
- 定义领域基础结构
- 与数据库表结构保持对应

### 7.5 DTO 层

DTO 层位于 `backend/internal/dto`。

职责：

- 定义请求结构
- 定义响应结构
- 定义分页响应、错误响应等公共结构

## 8. 功能模块边界

```text
官网内容：site
招新报名：recruit
管理后台：admin
用户系统：user / auth
博客社区：blog
社交关系：social
私信群聊：message / group
通知搜索：notification / search
文件上传：upload / storage
RAG：rag
最新消息：announcement
常用工具：tool
网站导航：navigation
一键关站：maintenance
审计日志：audit
```

## 9. 数据表规划

重构后建议保留已有社区表，同时将当前 JSON blob 拆成独立表。

建议新增或迁移表：

```text
news
awards
seniors
review_images
review_albums
recruit_applications
announcements
tool_categories
tools
nav_categories
nav_sites
site_settings
admin_audit_logs
rag_chat_sessions
rag_chat_messages
```

保留兼容：

```text
app_kv
app_cache
```

兼容用途：

- 读取旧 JSON 数据
- 做迁移兜底
- 保存少量系统配置
- 保存短期缓存

## 10. RAG 流式输出结构

后端新增：

```text
backend/internal/api/v1/rag_handler.go
backend/internal/service/rag_service.go
backend/internal/rag/stream.go
backend/internal/rag/chat.go
backend/internal/rag/retriever.go
```

前端新增：

```text
frontend/src/api/rag.ts
frontend/src/composables/useRagStream.ts
frontend/src/views/rag/RagChat.vue
```

推荐接口：

```text
POST /api/v1/rag/chat/stream
```

推荐事件：

```text
token
sources
done
error
```

## 11. 一键关站结构

后端新增：

```text
backend/internal/api/v1/maintenance_handler.go
backend/internal/service/maintenance_service.go
backend/internal/repository/maintenance_repository.go
backend/internal/model/maintenance.go
backend/internal/dto/maintenance_dto.go
backend/internal/middleware/maintenance.go
```

前端新增：

```text
frontend/src/api/maintenance.ts
frontend/src/stores/maintenance.ts
frontend/src/views/admin/MaintenanceManage.vue
frontend/src/views/maintenance/Maintenance.vue
```

数据存储：

```text
site_settings
admin_audit_logs
```

维护模式中间件应在进入业务路由前执行，并保留管理员登录、后台、健康检查、静态资源等白名单。

## 12. 迁移策略

建议分阶段迁移：

1. 创建 `backend/` 和 `frontend/` 新目录，旧代码保留。
2. 建立 Gin 启动骨架、配置、日志、中间件和健康检查。
3. 建立数据库迁移系统和 repository 基础层。
4. 迁移管理员登录和后台基础能力。
5. 迁移官网内容，将 `team_content` 拆表。
6. 迁移招新报名，将 `recruit_applications` 拆表。
7. 迁移上传和 RAG。
8. 迁移博客社区。
9. 新增最新消息、常用工具、网站导航。
10. 新增一键关站和审计日志。
11. 完成 Vue 前端替换。
12. 完成 Docker 部署和旧入口下线。

