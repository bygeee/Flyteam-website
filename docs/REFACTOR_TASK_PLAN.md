# Flyteam Website 重构与新增功能协作计划

## 1. 文档目标

本文档用于指导 6 人同时使用 Git 协作完成 Flyteam Website 的重构和新增功能开发。

本次重构目标包括：

- 后端改为 Go + Gin
- 后端拆分为 API 层、Service 层、Repository 层、Model 层、DTO 层
- 前端改为 Vue 3 + Vite
- 实现前后端分离
- 增加服务日志和后台审计日志
- 改为 Docker 部署
- 新增最新消息、常用工具、常用网站导航模块
- 新增 RAG 流式输出
- 新增一键关站功能
- 保留并迁移当前已有官网、招新、后台、RAG、博客社区功能

## 2. 协作原则

为了减少 Git 冲突，本次开发采用以下原则：

- 每个人拥有明确的目录所有权。
- 公共结构先约定，再并行开发。
- API、DTO、Model、数据库迁移必须优先评审。
- 旧代码在新功能验证前不删除。
- 每个阶段都要有可运行、可验证的交付物。
- SQL 只能写在 Repository 层。
- Vue 页面只通过 API 获取动态数据。
- Docker 部署必须使用 volume 保存数据库、上传文件和日志。

## 3. 推荐分支策略

主分支结构：

```text
main
└─ develop
   ├─ feat/backend-foundation
   ├─ feat/database-migration
   ├─ feat/site-recruit-api
   ├─ feat/community-rag-api
   ├─ feat/vue-public
   └─ feat/vue-admin-deploy
```

协作要求：

- 所有人从 `develop` 拉取功能分支。
- 不直接向 `main` 提交代码。
- 每个功能分支通过 Pull Request 合并到 `develop`。
- 涉及公共接口、数据库表、DTO 的改动必须先同步给其他成员。
- 每天至少同步一次 `develop`，避免长期分叉。

## 4. 6 人职责总览

| 成员 | 负责方向 | 主要目录所有权 | 主要产出 |
| --- | --- | --- | --- |
| A | 后端基础框架 | `backend/cmd`、`backend/internal/api`、`backend/internal/middleware`、`backend/internal/config`、`backend/internal/logger` | Gin 启动、路由、中间件、日志、统一响应 |
| B | 数据库与数据迁移 | `backend/internal/model`、`backend/internal/repository`、`backend/migrations`、`backend/scripts` | 表结构、Repository、旧 JSON 迁移 |
| C | 官网、招新和新增内容后端 | `site`、`recruit`、`announcement`、`tool`、`navigation`、`maintenance` 相关后端文件 | 官网内容、招新、最新消息、工具、导航、一键关站 API |
| D | 用户、博客、社区和 RAG 后端 | `user`、`blog`、`social`、`message`、`group`、`notification`、`rag` 相关后端文件 | 社区 API、博客 API、RAG 流式输出 |
| E | Vue 前台端 | `frontend/src/views/public`、`frontend/src/views/recruit`、`frontend/src/views/blog`、`frontend/src/views/user`、`frontend/src/views/rag`、`frontend/src/views/tools`、`frontend/src/views/navigation` | 官网、招新、博客、用户页、RAG 页面、工具导航页面 |
| F | Vue 后台与部署 | `frontend/src/views/admin`、`frontend/src/views/maintenance`、`deploy`、`Dockerfile`、`docker-compose.yml`、`docs` | 管理后台、维护页、Docker、Nginx、部署文档 |

## 5. 目录所有权约定

### 5.1 A 后端基础

A 主要维护：

```text
backend/cmd/server/main.go
backend/internal/api/router.go
backend/internal/middleware/
backend/internal/config/
backend/internal/logger/
backend/internal/util/response.go
```

A 应避免直接修改具体业务 service 和 repository。

### 5.2 B 数据库与迁移

B 主要维护：

```text
backend/internal/model/
backend/internal/repository/
backend/migrations/
backend/scripts/migrate_json_to_db.go
backend/scripts/seed_admin.go
```

B 负责保证数据库表结构、Repository 方法、迁移脚本稳定。

### 5.3 C 官网、招新和新增内容后端

C 主要维护：

```text
backend/internal/api/v1/site_handler.go
backend/internal/api/v1/recruit_handler.go
backend/internal/api/v1/announcement_handler.go
backend/internal/api/v1/tool_handler.go
backend/internal/api/v1/navigation_handler.go
backend/internal/api/v1/maintenance_handler.go

backend/internal/service/site_service.go
backend/internal/service/recruit_service.go
backend/internal/service/announcement_service.go
backend/internal/service/tool_service.go
backend/internal/service/navigation_service.go
backend/internal/service/maintenance_service.go
```

C 如需新增 Repository 方法，应先和 B 对齐。

### 5.4 D 用户、博客、社区和 RAG 后端

D 主要维护：

```text
backend/internal/api/v1/user_handler.go
backend/internal/api/v1/blog_handler.go
backend/internal/api/v1/social_handler.go
backend/internal/api/v1/message_handler.go
backend/internal/api/v1/group_handler.go
backend/internal/api/v1/notification_handler.go
backend/internal/api/v1/search_handler.go
backend/internal/api/v1/rag_handler.go

backend/internal/service/user_service.go
backend/internal/service/blog_service.go
backend/internal/service/social_service.go
backend/internal/service/message_service.go
backend/internal/service/group_service.go
backend/internal/service/notification_service.go
backend/internal/service/search_service.go
backend/internal/service/rag_service.go

backend/internal/rag/
```

D 负责现有博客社区行为兼容和 RAG 流式输出。

### 5.5 E Vue 前台端

E 主要维护：

```text
frontend/src/views/public/
frontend/src/views/recruit/
frontend/src/views/blog/
frontend/src/views/user/
frontend/src/views/social/
frontend/src/views/rag/
frontend/src/views/tools/
frontend/src/views/navigation/
frontend/src/components/public/
frontend/src/composables/useRagStream.ts
```

E 负责普通用户访问侧体验。

### 5.6 F Vue 后台与部署

F 主要维护：

```text
frontend/src/views/admin/
frontend/src/views/maintenance/
frontend/src/layouts/AdminLayout.vue
frontend/src/layouts/MaintenanceLayout.vue
deploy/
Dockerfile
docker-compose.yml
frontend/Dockerfile
backend/Dockerfile
```

F 负责后台管理闭环和部署闭环。

## 6. 阶段计划表

| 阶段 | 时间建议 | 参与人 | 完成内容 | 验收标准 |
| --- | --- | --- | --- | --- |
| 0. 项目冻结与接口约定 | 0.5-1 天 | 全员 | 确认重构范围、模块边界、API 命名、数据库表名、分支规范 | `docs/REFACTOR_REQUIREMENTS.md` 和 `docs/REFACTOR_ARCHITECTURE.md` 作为统一基线 |
| 1. 新目录骨架搭建 | 1-2 天 | A、B、F | 创建 `backend/`、`frontend/`、`deploy/` 基础结构，旧代码不删除 | 新旧目录并存，旧项目可继续运行 |
| 2. 后端 Gin 基础能力 | 2-3 天 | A | Gin 启动、`/api/v1/health`、统一响应、CORS、Recovery、RequestID、请求日志 | `go build` 通过，健康检查可访问 |
| 3. 数据库与迁移基础 | 3-5 天 | B | migrations、数据库连接、事务封装、model、repository 基类、旧 JSON 迁移脚本 | 可初始化 SQLite，迁移脚本可读取旧 JSON |
| 4. 官网与招新 API 迁移 | 4-6 天 | C、B | 新闻、奖项、前辈墙、回顾、招新报名从 JSON blob 拆表并提供 API | 官网内容 CRUD 可用，招新提交和后台管理可用 |
| 5. 新增消息、工具、导航 API | 3-4 天 | C | 最新消息、常用工具、常用网站导航模块 | 前台列表、详情、后台 CRUD API 可用 |
| 6. 用户、博客、社区 API 迁移 | 5-8 天 | D、B | 用户、文章、评论、点赞、收藏、关注、好友、私信、群聊、通知、搜索 | 现有社区功能 API 行为保持一致 |
| 7. RAG 流式输出 | 3-5 天 | D、A | 新增 SSE 流式接口、引用来源、错误事件、停止生成基础能力 | 前端可逐段接收回答，异常返回 `error` 事件 |
| 8. 一键关站 | 2-3 天 | A、C、F | 后端维护模式中间件、后台开关、维护页、审计日志 | 普通访问返回维护页，API 返回 503，后台仍可登录恢复 |
| 9. Vue 前台重构 | 6-10 天 | E | 官网、招新、博客、用户中心、RAG、工具、导航页面 | 前台主要页面通过 Vue 可访问，数据来自 API |
| 10. Vue 后台重构 | 6-10 天 | F、C、D | 登录、Dashboard、内容管理、招新管理、用户管理、RAG 管理、关站管理 | 后台管理流程闭环 |
| 11. Docker 部署 | 2-4 天 | F、A | 后端 Dockerfile、前端 Dockerfile、Nginx、Compose、volume | `docker compose up --build` 可启动完整系统 |
| 12. 联调与收尾 | 4-7 天 | 全员 | API 联调、权限测试、迁移验证、日志审计、文档补齐 | 核心功能全部通过验收，旧入口可准备下线 |

## 7. 成员具体任务清单

### 7.1 A 后端基础框架

第一批任务：

- 创建 Gin 服务启动入口。
- 创建 `/api/v1/health`。
- 建立统一响应格式。
- 建立统一错误处理。
- 建立 CORS 中间件。
- 建立 Recovery 中间件。
- 建立 RequestID 中间件。
- 建立请求日志中间件。

第二批任务：

- 接入管理员鉴权中间件。
- 接入普通用户鉴权中间件。
- 接入 CSRF 校验。
- 接入维护模式中间件。
- 接入限流中间件。
- 与 F 联调 Docker 后端启动。

最终交付：

- 稳定的 Gin 后端基础框架。
- 所有业务模块可以挂载到 `/api/v1` 下。
- 具备统一日志、统一响应、统一错误处理。

### 7.2 B 数据库与迁移

第一批任务：

- 设计 migrations 执行方式。
- 创建基础表结构。
- 建立数据库连接封装。
- 建立事务封装。
- 建立 model 规范。
- 建立 repository 规范。

第二批任务：

- 将 `team_content` 拆分为新闻、奖项、前辈、回顾等表。
- 将 `recruit_applications` 拆分为独立表。
- 新增最新消息、工具、导航、维护状态、审计日志表。
- 编写旧 JSON 到新表的迁移脚本。

最终交付：

- 清晰可扩展的数据层。
- 旧数据可迁移。
- 业务层不直接接触 SQL。

### 7.3 C 官网、招新和新增内容后端

第一批任务：

- 迁移官网内容读取 API。
- 迁移官网内容管理 API。
- 迁移招新验证码。
- 迁移招新提交。
- 迁移招新后台管理。

第二批任务：

- 新增最新消息模块 API。
- 新增常用工具模块 API。
- 新增常用网站导航模块 API。
- 新增一键关站状态 API。
- 对接审计日志。

最终交付：

- 官网、招新、新增内容模块全部通过 Gin API 提供。
- 后台可管理官网内容、招新、消息、工具、导航和关站状态。

### 7.4 D 用户、博客、社区和 RAG 后端

第一批任务：

- 迁移用户注册、登录、退出。
- 迁移用户资料和密码修改。
- 迁移博客文章列表、详情、创建、编辑、删除、发布。
- 迁移评论、点赞、收藏。

第二批任务：

- 迁移关注、好友、私信、群聊、通知、搜索。
- 迁移头像和博客图片上传逻辑。
- 新增 RAG SSE 流式输出。
- 新增 RAG 引用来源事件。
- 新增 RAG 错误事件。

最终交付：

- 现有社区和博客功能完整迁移。
- RAG 支持流式输出。
- API 行为与旧系统兼容。

### 7.5 E Vue 前台端

第一批任务：

- 初始化 Vue 3 + Vite。
- 建立 Vue Router。
- 建立 Pinia 状态管理。
- 建立前台公共布局。
- 封装前台 API 请求。
- 重构首页、介绍、新闻、奖项、回顾、前辈墙。

第二批任务：

- 重构招新报名页面。
- 重构博客首页和文章详情。
- 重构用户登录、注册、用户中心、个人主页。
- 重构私信、群聊、通知。
- 重构 RAG 流式问答页面。
- 新增常用工具和网站导航页面。

最终交付：

- 普通用户访问侧页面全部 Vue 化。
- 所有动态数据来自 API。
- RAG 页面支持流式显示。

### 7.6 F Vue 后台与部署

第一批任务：

- 建立后台布局。
- 建立后台登录页。
- 建立 Dashboard。
- 建立后台路由守卫。
- 建立后台 API 请求封装。

第二批任务：

- 重构官网内容管理。
- 重构招新管理。
- 重构用户管理。
- 重构博客审核。
- 新增最新消息管理。
- 新增工具管理。
- 新增导航管理。
- 新增一键关站管理。
- 新增审计日志页面。

部署任务：

- 编写后端 Dockerfile。
- 编写前端 Dockerfile。
- 编写 Nginx 配置。
- 编写 docker-compose.yml。
- 配置 storage volume。
- 编写部署文档。

最终交付：

- 管理端完整可用。
- Docker 一键启动完整系统。
- 生产部署路径清晰。

## 8. 关键里程碑

| 里程碑 | 标准 |
| --- | --- |
| M1 后端骨架完成 | Gin 启动、健康检查、日志、中间件可用 |
| M2 数据层完成 | 新表可创建，旧 JSON 可迁移 |
| M3 官网 / 招新 API 完成 | 原官网内容和招新功能可通过 API 使用 |
| M4 社区 / RAG API 完成 | 博客社区功能恢复，RAG 支持流式输出 |
| M5 Vue 前台可用 | 普通用户主要页面迁移完成 |
| M6 Vue 后台可用 | 管理员可完成内容、用户、关站管理 |
| M7 Docker 可部署 | Compose 一键启动，数据持久化 |
| M8 重构验收 | 旧功能不丢，新功能可用，文档完整 |

## 9. 优先级建议

### 9.1 第一优先级

- 后端 Gin 基础
- 数据库迁移
- 官网内容 API
- 招新 API
- 后台登录
- Docker 基础启动

这些内容决定系统能否进入可并行开发状态。

### 9.2 第二优先级

- Vue 官网
- Vue 后台
- 最新消息
- 常用工具
- 网站导航
- 一键关站

这些内容是本次重构和新增功能的主要用户可见成果。

### 9.3 第三优先级

- 社区完整迁移
- RAG 流式输出优化
- 审计日志完善
- 旧入口下线

这些内容依赖基础框架、数据层和前端基础稳定后推进。

## 10. 每阶段交付要求

每个阶段完成时，负责成员需要提供：

- 已完成内容列表
- 修改的主要文件
- 新增或调整的 API
- 新增或调整的数据表
- 本地验证命令
- 已知问题
- 后续依赖

建议 Pull Request 描述格式：

```text
## 完成内容

## 修改范围

## 验证方式

## 风险点

## 需要其他成员关注
```

## 11. 联调检查清单

后端联调检查：

- `/api/v1/health` 正常。
- 登录接口正常。
- 权限中间件能拦截未登录请求。
- 维护模式能拦截普通请求。
- API 返回格式统一。
- 错误响应格式统一。
- 请求日志包含 `request_id`。

数据库联调检查：

- migrations 可重复执行。
- 旧 JSON 数据可迁移。
- 新表能支持分页查询。
- Repository 单元测试通过。
- 事务回滚正常。

前端联调检查：

- API base URL 可配置。
- 登录态刷新后仍可恢复。
- 前台路由刷新不 404。
- 后台路由守卫正常。
- RAG 流式输出正常。
- 维护页展示正常。

部署联调检查：

- `docker compose up --build` 可启动。
- 前端可访问。
- `/api` 可反代到后端。
- 上传文件写入 volume。
- SQLite 数据重启后不丢失。
- 日志写入 `storage/logs`。

## 12. 风险与处理

| 风险 | 影响 | 处理方式 |
| --- | --- | --- |
| 多人同时修改公共 DTO 或 Model | 容易冲突 | 公共结构先 PR，合并后其他人同步 |
| 旧 JSON 数据迁移不完整 | 线上内容丢失 | 保留旧 JSON 读取能力，迁移前备份 |
| 前后端接口不一致 | 联调成本高 | 先维护 `docs/API_DESIGN.md`，再开发 |
| Gin 重构范围过大 | 容易半成品 | 按模块迁移，新旧代码并存 |
| Docker volume 配置错误 | 数据丢失 | 本地和生产分别验证持久化 |
| 一键关站误锁后台 | 无法恢复 | 后台登录、后台 API、健康检查必须白名单 |
| RAG 流式输出异常 | 前端体验差 | SSE 定义 `token`、`sources`、`done`、`error` 四类事件 |

## 13. 最终验收标准

最终完成后应满足：

- 后端已迁移到 Gin。
- 前端已迁移到 Vue 3。
- 前后端完全通过 API 通信。
- 核心业务 SQL 位于 Repository 层。
- 官网、招新、后台、RAG、博客社区功能不丢失。
- 最新消息、常用工具、网站导航可用。
- 一键关站可用且不会锁死管理员。
- RAG 支持流式输出。
- Docker Compose 可启动完整系统。
- 上传文件、数据库、日志可持久化。
- 文档完整，后续成员可继续开发。

