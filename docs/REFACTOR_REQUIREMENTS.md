# Flyteam Website 重构需求文档

## 1. 文档目标

本文档用于梳理当前 Flyteam Website 已实现的功能，以及后续重构和新增功能的具体需求。

当前项目已经具备团队官网、管理后台、招新报名、RAG 知识库问答、博客社区等能力。后续重构目标不是简单调整目录，而是将项目升级为前后端分离、后端分层清晰、可 Docker 部署、可持续扩展的系统。

## 2. 已实现功能

### 2.1 官网展示

当前已实现公开官网页面：

- 首页
- 团队介绍
- 团队新闻
- 奖项荣誉
- 团队回顾
- 回顾相册
- Flyteamers / 前辈墙
- 招新页面

当前已实现能力：

- 首页图片展示
- 团队简介和团队概览展示
- 新闻列表和新闻详情
- 奖项荣誉展示
- 团队回顾图片展示
- 回顾相册展示
- 前辈墙展示
- 公开内容接口读取

当前主要数据来源是 `team_content`。该数据以 JSON 形式存储在 SQLite 的 `app_kv` 表中，并兼容旧文件 `storage/team_content.json`。

### 2.2 招新报名

当前已实现：

- 招新报名表单
- C 语言输出结果验证码
- 报名提交
- 报名列表查看
- 报名统计
- 报名信息更新
- 报名信息删除
- 招新方向 / hall 分类

当前招新报名数据存储在 `recruit_applications` JSON 中，位于 SQLite 的 `app_kv` 表，并兼容旧文件 `storage/recruit_applications.json`。

### 2.3 管理后台

当前已实现：

- 管理员登录
- 管理员退出
- 管理员会话
- CSRF 校验
- 管理员账号管理
- 管理员角色区分
- 超级管理员相关能力
- 官网内容管理
- 招新报名管理
- 社区用户审核和管理

管理员账号已经有独立 SQLite 表 `admin_users`，并兼容旧文件 `storage/admin_users.json`。

### 2.4 文件上传

当前已实现上传类型：

- PDF 知识库文件
- 首页图片
- 奖项图片
- 前辈照片
- 回顾图片
- 新闻图片
- 博客图片
- 用户头像

上传文件存储在 `storage/uploads/` 下，数据库或 JSON 中保存文件 URL。

### 2.5 RAG 知识库问答

当前已实现：

- PDF 上传
- 默认知识库导入
- 本地文件导入
- 知识库重建
- PDF 文本提取
- 文本切块
- Embedding 检索
- 调用 OpenAI-compatible Chat API
- 普通非流式问答

当前 RAG 索引优先存储在 SQLite 的 `app_cache` 表中，并兼容 `app_kv.rag_index` 和旧文件 `storage/rag_index_go.json`。

### 2.6 普通用户与博客社区

当前已实现：

- 用户注册
- 用户登录
- 用户退出
- 用户资料查看和修改
- 密码修改
- 头像上传
- 公开用户主页
- 文章列表
- 文章详情
- 文章创建
- 文章编辑
- 文章删除
- 文章发布
- 文章浏览量
- 文章推荐
- 评论
- 点赞
- 收藏

博客和社区核心数据已使用 SQLite 关系表存储，例如：

- `community_users`
- `community_sessions`
- `blog_articles`
- `blog_article_tags`
- `blog_article_versions`
- `blog_comments`
- `blog_likes`
- `blog_favorites`

### 2.7 社交、私信、群聊、通知

当前已实现：

- 关注 / 取关
- 粉丝列表
- 关注列表
- 好友申请
- 好友接受
- 好友拒绝
- 好友删除
- 私信会话
- 私信消息
- 群聊创建
- 群资料管理
- 群成员管理
- 群消息
- 站内通知
- 全站搜索

这些模块已使用 SQLite 表存储，例如：

- `social_follows`
- `friend_requests`
- `friendships`
- `private_conversations`
- `private_messages`
- `chat_groups`
- `chat_group_members`
- `chat_group_messages`
- `notifications`

### 2.8 当前技术形态

当前技术实现：

- 后端：Go 标准库 `net/http`
- 路由：自定义 `ServeHTTP` 和 route 分发
- 前端：静态 HTML / CSS / JS
- 数据库：SQLite + JSON blob + 旧 JSON 文件兼容
- 部署：Go 程序直接运行

当前未使用 Gin，未使用 Vue，未形成 Docker 标准部署结构。

## 3. 需要添加和重构的功能

### 3.1 后端架构重构

目标是把当前 `internal/app` 中混合的 HTTP、业务逻辑、SQL、上传、RAG、权限逻辑拆开。

新增要求：

- 后端使用 Gin 框架
- 统一 API 前缀为 `/api/v1`
- 拆分 API 层、Service 层、Repository 层、Model 层、DTO 层
- SQL 只能出现在 Repository 层
- Handler 只负责请求解析、权限检查、响应返回
- Service 负责业务流程编排
- Model 负责数据库实体
- DTO 负责请求和响应结构
- Middleware 负责日志、鉴权、CORS、恢复、限流和全局维护模式

建议后端结构：

```text
backend/internal/api
backend/internal/service
backend/internal/repository
backend/internal/model
backend/internal/dto
backend/internal/middleware
backend/internal/config
backend/internal/logger
```

验收标准：

- Gin 服务可启动
- `/api/v1/health` 可访问
- 旧功能逐步迁移后行为保持一致
- 业务代码不再直接在 handler 中写 SQL

### 3.2 前后端分离

目标是后端只提供 API，前端完全独立。

新增要求：

- 后端不再负责渲染 HTML 页面
- 前端通过 API 获取所有动态数据
- 前端构建产物由 Nginx 或独立静态服务提供
- 上传文件继续通过后端 API 管理
- API 地址可通过前端环境变量配置

验收标准：

- 前端独立运行
- 后端独立运行
- 前端 API 地址可配置
- 刷新 Vue 路由页面不 404

### 3.3 Vue 前端重构

目标是用 Vue 3 重建当前静态页面。

新增要求：

- 使用 Vue 3 + Vite
- 使用 Vue Router 管理路由
- 使用 Pinia 管理登录态、用户态、后台状态
- 封装统一 API 请求模块
- 重构官网页面
- 重构管理后台
- 重构招新页面
- 重构博客社区页面
- 重构 RAG 问答页面
- 重构维护页

建议前端页面分区：

- `public`：官网展示
- `admin`：管理后台
- `user`：用户中心
- `blog`：博客社区
- `social`：私信、群聊、通知
- `rag`：知识库问答
- `tools`：常用工具
- `navigation`：常用网站导航
- `maintenance`：维护页

### 3.4 Gin 服务日志

新增服务日志能力：

- 请求日志
- 错误日志
- panic 恢复日志
- 管理员操作审计日志
- 文件上传日志
- RAG 调用日志

日志字段至少包括：

- `request_id`
- `method`
- `path`
- `status`
- `latency`
- `client_ip`
- `user_agent`
- `user_id`
- `admin_id`
- `error`
- `created_at`

后台高危操作必须记录审计日志：

- 登录后台
- 新增 / 修改 / 删除内容
- 上传文件
- 删除文件
- 封禁用户
- 恢复用户
- 关站 / 开站
- 重建 RAG 索引

### 3.5 Docker 部署

新增要求：

- 后端 `Dockerfile`
- 前端 `Dockerfile`
- `docker-compose.yml`
- 生产 Nginx 配置
- `.env.example` 更新
- `storage` 使用 volume 挂载

容器划分：

- `frontend`：Vue build 后由 Nginx 提供
- `backend`：Gin API 服务
- `storage`：挂载 SQLite、上传文件、日志、RAG 索引

验收标准：

- `docker compose up --build` 可启动完整项目
- `/api` 反向代理到后端
- 前端路由可刷新
- `storage` 数据重启后不丢失

### 3.6 最新消息模块

新增“最新消息 / 公告”模块。

功能要求：

- 前台展示最新消息
- 支持消息详情
- 支持置顶
- 支持分类
- 支持封面图
- 支持发布时间
- 支持启用 / 隐藏
- 后台可新增、编辑、删除

建议数据表：

```text
announcements
```

核心字段：

```text
id
title
summary
content
category
cover_url
pinned
visible
published_at
created_at
updated_at
```

### 3.7 常用工具模块

新增“常用工具”页面。

功能要求：

- 工具分类
- 工具名称
- 工具简介
- 工具图标
- 工具链接
- 排序
- 启用 / 禁用
- 点击量统计
- 后台 CRUD

建议数据表：

- `tool_categories`
- `tools`

适用内容：

- 报名入口
- RAG 问答
- 文档下载
- 比赛资料
- 代码格式化工具
- 内部系统入口

### 3.8 常用网站导航模块

新增“常用网站导航”页面。

功能要求：

- 网站分类
- 网站名称
- 网站 URL
- 网站简介
- 网站图标
- 推荐标记
- 排序
- 启用 / 禁用
- 点击量统计
- 后台 CRUD

建议数据表：

- `nav_categories`
- `nav_sites`

适用内容：

- 竞赛平台
- 学习网站
- 学校系统
- GitHub 仓库
- 官方文档
- 安全工具网站

### 3.9 RAG 流式输出

当前 RAG 是普通 JSON 响应，需要新增流式输出。

新增接口：

```text
POST /api/v1/rag/chat/stream
```

推荐使用 SSE：

```text
event: token
data: {"content":"..."}

event: sources
data: {"documents":[...]}

event: done
data: {"message_id":"..."}

event: error
data: {"message":"..."}
```

功能要求：

- 前端实时显示回答内容
- 支持显示引用来源
- 支持停止生成
- 支持失败提示
- 支持历史记录
- 服务端逐段转发模型输出

验收标准：

- 用户提交问题后无需等待完整回答
- 回答内容逐字或分段出现
- 结束时返回引用来源
- 异常时返回明确错误事件

### 3.10 一键关站功能

新增全局维护模式，不等同于当前博客站开关。

功能要求：

- 管理员可一键关闭全站
- 管理员可一键恢复开站
- 可填写维护公告
- 可填写关站原因
- 可填写预计恢复时间
- 普通用户访问页面时进入维护页
- 普通 API 返回 `503 Service Unavailable`
- 管理员后台和登录页仍可访问

白名单建议：

- `/login`
- `/admin`
- `/api/v1/admin/*`
- `/api/v1/health`
- `/static/*`
- 维护页所需资源

关站状态建议存储为 `site_global_state`，核心字段：

```text
closed
notice
reason
expected_restore_at
updated_by
updated_at
```

验收标准：

- 关站后普通用户无法访问官网、招新、博客、RAG
- API 返回 `503 Service Unavailable`
- 管理员仍能登录后台
- 开站后访问恢复正常
- 关站 / 开站操作写入审计日志
- 服务重启后关站状态仍然有效

### 3.11 数据存储重构

当前 `team_content` 和 `recruit_applications` 还是 JSON blob，后续应拆成独立表。

建议新增或迁移为：

- `news`
- `awards`
- `seniors`
- `review_images`
- `review_albums`
- `recruit_applications`
- `announcements`
- `tools`
- `tool_categories`
- `nav_sites`
- `nav_categories`
- `site_settings`
- `admin_audit_logs`

迁移要求：

- 保留旧 JSON 读取能力
- 提供一次性迁移脚本
- 迁移后新写入走数据表
- 避免破坏现有线上数据

## 4. 总体验收目标

最终项目应达到：

- 后端 Gin 化
- 前后端分离
- Vue 前端可独立开发
- Docker 一键部署
- 核心数据表结构清晰
- 日志和审计可追踪
- RAG 支持流式输出
- 新增最新消息模块
- 新增常用工具模块
- 新增常用网站导航模块
- 支持全局一键关站
- 现有官网、招新、后台、博客社区功能不丢失

