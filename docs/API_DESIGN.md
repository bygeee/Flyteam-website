# Flyteam Website API 接口设计文档

## 1. 文档目标

本文档用于约定 Flyteam Website 重构后的 API 设计，供后端、前端、测试和部署协作使用。

当前旧项目使用无版本前缀的 `/api/*` 接口。重构后统一使用：

```text
/api/v1
```

后续 Vue 前端只通过本文档中的 API 与后端通信。

## 2. 通用约定

### 2.1 Base URL

本地开发：

```text
http://127.0.0.1:8000/api/v1
```

Docker / Nginx 反向代理后：

```text
/api/v1
```

### 2.2 Content-Type

普通 JSON 请求：

```http
Content-Type: application/json
```

文件上传：

```http
Content-Type: multipart/form-data
```

RAG 流式输出：

```http
Accept: text/event-stream
```

### 2.3 统一响应格式

成功响应：

```json
{
  "success": true,
  "data": {},
  "message": "ok",
  "request_id": "req_xxx"
}
```

失败响应：

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "参数不合法"
  },
  "request_id": "req_xxx"
}
```

分页响应：

```json
{
  "success": true,
  "data": {
    "items": [],
    "page": 1,
    "page_size": 20,
    "total": 100
  },
  "request_id": "req_xxx"
}
```

### 2.4 分页参数

通用分页 Query 参数：

```text
page=1
page_size=20
q=keyword
sort=created_at_desc
```

建议限制：

- `page` 最小为 `1`
- `page_size` 默认 `20`
- `page_size` 最大 `100`

### 2.5 认证方式

管理员认证：

```http
Cookie: admin_session=xxx
X-CSRF-Token: xxx
```

或兼容开发调试：

```http
X-Admin-Token: xxx
```

普通用户认证：

```http
Cookie: user_session=xxx
X-CSRF-Token: xxx
```

或兼容开发调试：

```http
X-User-Token: xxx
```

说明：

- Cookie 会话用于浏览器端。
- Header Token 可用于开发调试、接口测试和脚本调用。
- 使用 Cookie 且请求方法为 POST、PUT、PATCH、DELETE 时必须校验 CSRF。

### 2.6 常见状态码

| 状态码 | 含义 |
| --- | --- |
| 200 | 请求成功 |
| 201 | 创建成功 |
| 204 | 删除成功且无返回体 |
| 400 | 请求参数错误 |
| 401 | 未登录 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 409 | 数据冲突 |
| 422 | 业务校验失败 |
| 429 | 请求过于频繁 |
| 500 | 服务端错误 |
| 503 | 站点维护中 |

### 2.7 一键关站响应

当全站维护模式开启时，普通 API 返回：

```http
HTTP/1.1 503 Service Unavailable
Retry-After: 3600
```

```json
{
  "success": false,
  "error": {
    "code": "SITE_MAINTENANCE",
    "message": "网站维护中，请稍后再试。"
  },
  "data": {
    "maintenance": true,
    "notice": "网站维护中，请稍后再试。",
    "expected_restore_at": "2026-05-11T10:00:00+08:00"
  },
  "request_id": "req_xxx"
}
```

## 3. 系统与健康检查

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| GET | `/health` | 公开 | 健康检查 |
| GET | `/status` | 公开 | 系统状态，包含 RAG 是否可用 |

示例：

```http
GET /api/v1/health
```

```json
{
  "success": true,
  "data": {
    "status": "ok",
    "version": "v1",
    "time": "2026-05-10T21:00:00+08:00"
  }
}
```

## 4. 管理员认证与账号

### 4.1 认证接口

| 方法 | 路径 | 权限 | 说明 | 旧接口 |
| --- | --- | --- | --- | --- |
| POST | `/admin/auth/login` | 公开 | 管理员登录 | `POST /api/admin/login` |
| POST | `/admin/auth/logout` | 管理员 | 管理员退出 | `POST /api/admin/logout` |
| GET | `/admin/auth/me` | 管理员 | 获取当前管理员 | `GET /api/admin/ping` |

登录请求：

```json
{
  "username": "admin",
  "password": "admin123456"
}
```

登录响应：

```json
{
  "success": true,
  "data": {
    "admin": {
      "id": "adm_xxx",
      "username": "admin",
      "display_name": "System Admin",
      "role": "site_admin"
    },
    "csrf_token": "csrf_xxx"
  }
}
```

### 4.2 管理员账号管理

| 方法 | 路径 | 权限 | 说明 | 旧接口 |
| --- | --- | --- | --- | --- |
| GET | `/admin/users` | 管理员 | 管理员列表 | `GET /api/admin/users` |
| POST | `/admin/users` | 超级管理员 | 新增管理员 | `POST /api/admin/users` |
| PUT | `/admin/users/{id}/password` | 超级管理员 | 修改管理员密码 | `PUT /api/admin/users/{id}/password` |
| PUT | `/admin/users/{id}/role` | 超级管理员 | 修改管理员角色 | `PUT /api/admin/users/{id}/role` |
| DELETE | `/admin/users/{id}` | 超级管理员 | 删除管理员 | `DELETE /api/admin/users/{id}` |

## 5. 官网内容接口

### 5.1 公开读取

| 方法 | 路径 | 权限 | 说明 | 旧接口 |
| --- | --- | --- | --- | --- |
| GET | `/site/content` | 公开 | 官网聚合内容 | `GET /api/content` |
| GET | `/site/intro` | 公开 | 团队介绍 | `GET /api/content` |
| GET | `/site/overview` | 公开 | 团队概览 | `GET /api/content` |
| GET | `/news` | 公开 | 新闻列表 | `GET /api/content` |
| GET | `/news/{id}` | 公开 | 新闻详情 | `GET /api/news/{id}` |
| GET | `/awards` | 公开 | 奖项列表 | `GET /api/content` |
| GET | `/seniors` | 公开 | 前辈墙列表 | `GET /api/content` |
| GET | `/reviews` | 公开 | 回顾图片列表 | `GET /api/content` |
| GET | `/review-albums` | 公开 | 回顾相册列表 | `GET /api/content` |
| GET | `/review-albums/{id}` | 公开 | 回顾相册详情 | `GET /api/review/albums/{id}` |

### 5.2 后台管理

| 方法 | 路径 | 权限 | 说明 | 旧接口 |
| --- | --- | --- | --- | --- |
| POST | `/admin/site/intro` | 管理员 | 保存团队介绍 | `POST /api/content/intro` |
| POST | `/admin/site/overview` | 管理员 | 保存团队概览 | `POST /api/content/overview` |
| POST | `/admin/news` | 管理员 | 新增新闻 | `POST /api/news` |
| PUT | `/admin/news/{id}` | 管理员 | 修改新闻 | `PUT /api/news/{id}` |
| DELETE | `/admin/news/{id}` | 管理员 | 删除新闻 | `DELETE /api/news/{id}` |
| POST | `/admin/awards` | 管理员 | 新增奖项 | `POST /api/awards` |
| PUT | `/admin/awards/{id}` | 管理员 | 修改奖项 | `PUT /api/awards/{id}` |
| DELETE | `/admin/awards/{id}` | 管理员 | 删除奖项 | `DELETE /api/awards/{id}` |
| POST | `/admin/seniors` | 管理员 | 新增前辈 | `POST /api/seniors` |
| PUT | `/admin/seniors/{id}` | 管理员 | 修改前辈 | `PUT /api/seniors/{id}` |
| DELETE | `/admin/seniors/{id}` | 管理员 | 删除前辈 | `DELETE /api/seniors/{id}` |
| POST | `/admin/review-images` | 管理员 | 新增回顾图片 | `POST /api/review` |
| PUT | `/admin/review-images/{id}` | 管理员 | 修改回顾图片 | `PUT /api/review/{id}` |
| DELETE | `/admin/review-images/{id}` | 管理员 | 删除回顾图片 | `DELETE /api/review/{id}` |
| POST | `/admin/review-albums` | 管理员 | 新增回顾相册 | `POST /api/review/albums` |
| PUT | `/admin/review-albums/{id}` | 管理员 | 修改回顾相册 | `PUT /api/review/albums/{id}` |
| DELETE | `/admin/review-albums/{id}` | 管理员 | 删除回顾相册 | `DELETE /api/review/albums/{id}` |
| POST | `/admin/review-albums/{id}/images/delete` | 管理员 | 删除相册图片 | `POST /api/review/albums/{id}/image/delete` |

## 6. 招新报名接口

### 6.1 公开接口

| 方法 | 路径 | 权限 | 说明 | 旧接口 |
| --- | --- | --- | --- | --- |
| GET | `/recruit/halls` | 公开 | 招新方向列表 | `GET /api/recruit/halls` |
| GET | `/recruit/captcha` | 公开 | 获取 C 语言验证码 | `GET /api/recruit/captcha` |
| GET | `/recruit/stats` | 公开 | 招新报名统计 | `GET /api/recruit/stats` |
| POST | `/recruit/applications` | 公开 | 提交报名 | `POST /api/recruit/apply` |

报名请求：

```json
{
  "name": "张三",
  "student_id": "20260001",
  "college": "计算机学院",
  "grade": "2026",
  "phone": "13800000000",
  "wechat": "flyteam",
  "email": "student@example.com",
  "hall": "web",
  "direction_detail": "Web 安全",
  "experience": "有 CTF 经历",
  "weekly_hours": "10",
  "note": "备注",
  "captcha_token": "cap_xxx",
  "captcha_answer": "42"
}
```

### 6.2 后台管理

| 方法 | 路径 | 权限 | 说明 | 旧接口 |
| --- | --- | --- | --- | --- |
| GET | `/admin/recruit/applications` | 管理员 | 报名列表 | `GET /api/recruit/list` |
| PUT | `/admin/recruit/applications/{id}` | 管理员 | 更新报名 | `PUT /api/recruit/{id}` |
| DELETE | `/admin/recruit/applications/{id}` | 管理员 | 删除报名 | `DELETE /api/recruit/{id}` |

## 7. 文件上传接口

| 方法 | 路径 | 权限 | 说明 | 旧接口 |
| --- | --- | --- | --- | --- |
| POST | `/uploads/pdf` | 管理员 | 上传 RAG PDF | `POST /api/upload` |
| POST | `/uploads/images` | 管理员 | 上传首页/通用图片 | `POST /api/upload/images` |
| POST | `/uploads/awards/images` | 管理员 | 上传奖项图片 | `POST /api/upload/awards/images` |
| POST | `/uploads/seniors/images` | 管理员 | 上传前辈照片 | `POST /api/upload/seniors/images` |
| POST | `/uploads/review/images` | 管理员 | 上传回顾图片 | `POST /api/upload/review/images` |
| POST | `/uploads/news/images` | 管理员 | 上传新闻图片 | `POST /api/upload/news/images` |
| POST | `/uploads/blog/images` | 用户 | 上传博客图片 | `POST /api/upload/blog/images` |
| POST | `/uploads/avatar` | 用户 | 上传用户头像 | `POST /api/upload/avatar` |

上传响应：

```json
{
  "success": true,
  "data": {
    "urls": [
      "/uploads/news/xxx.jpg"
    ]
  }
}
```

## 8. RAG 知识库接口

| 方法 | 路径 | 权限 | 说明 | 旧接口 |
| --- | --- | --- | --- | --- |
| GET | `/rag/status` | 公开 | RAG 状态 | `GET /api/status` |
| POST | `/rag/ingest/default` | 管理员 | 导入默认知识库 | `POST /api/ingest/default` |
| POST | `/rag/ingest/rebuild-default` | 管理员 | 重建默认知识库 | `POST /api/ingest/rebuild/default` |
| POST | `/rag/ingest/local` | 管理员 | 导入本地文件 | `POST /api/ingest/local` |
| POST | `/rag/chat` | 公开/用户 | 普通非流式问答 | `POST /api/chat` |
| POST | `/rag/chat/stream` | 公开/用户 | SSE 流式问答 | 新增 |

普通问答请求：

```json
{
  "question": "Flyteam 是什么？"
}
```

流式问答请求：

```json
{
  "question": "Flyteam 的招新方向有哪些？",
  "conversation_id": "optional",
  "top_k": 5
}
```

SSE 事件：

```text
event: token
data: {"content":"Flyteam"}

event: sources
data: {"documents":[{"title":"Flyteam.pdf","score":0.83}]}

event: done
data: {"message_id":"msg_xxx"}

event: error
data: {"message":"模型调用失败"}
```

## 9. 普通用户接口

### 9.1 用户认证

| 方法 | 路径 | 权限 | 说明 | 旧接口 |
| --- | --- | --- | --- | --- |
| POST | `/auth/register` | 公开 | 用户注册 | `POST /api/users/register` |
| POST | `/auth/login` | 公开 | 用户登录 | `POST /api/users/login` |
| POST | `/auth/logout` | 用户 | 用户退出 | `POST /api/users/logout` |
| GET | `/users/me` | 用户 | 当前用户资料 | `GET /api/users/me` |
| PUT | `/users/me/settings` | 用户 | 修改个人设置 | `PUT /api/users/me/settings` |
| PUT | `/users/me/password` | 用户 | 修改密码 | `PUT /api/users/me/password` |

注册请求：

```json
{
  "user_id": "alice",
  "nickname": "Alice",
  "password": "password123"
}
```

登录请求：

```json
{
  "user_id": "alice",
  "password": "password123"
}
```

### 9.2 用户主页

| 方法 | 路径 | 权限 | 说明 | 旧接口 |
| --- | --- | --- | --- | --- |
| GET | `/users/{id}` | 公开 | 公开用户主页 | `GET /api/users/{id}` |
| PUT | `/users/{id}` | 本人/管理员 | 编辑用户资料 | `PUT /api/users/{id}` |

## 10. 博客接口

| 方法 | 路径 | 权限 | 说明 | 旧接口 |
| --- | --- | --- | --- | --- |
| GET | `/blog/articles` | 公开 | 文章列表 | `GET /api/blog/articles` |
| POST | `/blog/articles` | 用户 | 创建文章 | `POST /api/blog/articles` |
| GET | `/blog/articles/{id}` | 公开 | 文章详情 | `GET /api/blog/articles/{id}` |
| PUT | `/blog/articles/{id}` | 作者/管理员 | 修改文章 | `PUT /api/blog/articles/{id}` |
| DELETE | `/blog/articles/{id}` | 作者/管理员 | 删除文章 | `DELETE /api/blog/articles/{id}` |
| POST | `/blog/articles/{id}/publish` | 作者/管理员 | 发布文章 | `POST /api/blog/articles/{id}/publish` |
| POST | `/blog/articles/{id}/view` | 公开 | 记录浏览 | `POST /api/blog/articles/{id}/view` |
| GET | `/blog/recommendations` | 公开 | 推荐文章 | `GET /api/blog/recommendations` |
| GET | `/blog/articles/{id}/comments` | 公开 | 评论列表 | `GET /api/blog/articles/{id}/comments` |
| POST | `/blog/articles/{id}/comments` | 用户 | 发表评论 | `POST /api/blog/articles/{id}/comments` |
| PUT | `/blog/comments/{id}` | 作者/管理员 | 修改评论 | `PUT /api/blog/comments/{id}` |
| DELETE | `/blog/comments/{id}` | 作者/管理员 | 删除评论 | `DELETE /api/blog/comments/{id}` |
| POST | `/blog/articles/{id}/like` | 用户 | 点赞 | `POST /api/blog/articles/{id}/like` |
| DELETE | `/blog/articles/{id}/like` | 用户 | 取消点赞 | `DELETE /api/blog/articles/{id}/like` |
| POST | `/blog/articles/{id}/favorite` | 用户 | 收藏 | `POST /api/blog/articles/{id}/favorite` |
| DELETE | `/blog/articles/{id}/favorite` | 用户 | 取消收藏 | `DELETE /api/blog/articles/{id}/favorite` |

文章创建请求：

```json
{
  "title": "文章标题",
  "summary": "文章摘要",
  "cover_url": "/uploads/blog/cover.jpg",
  "content_markdown": "# 正文",
  "tags": ["Go", "CTF"],
  "category": "技术",
  "language": "zh-CN",
  "status": "draft"
}
```

## 11. 社交关系接口

| 方法 | 路径 | 权限 | 说明 | 旧接口 |
| --- | --- | --- | --- | --- |
| POST | `/social/follows/{id}` | 用户 | 关注用户 | `POST /api/social/follows/{id}` |
| DELETE | `/social/follows/{id}` | 用户 | 取消关注 | `DELETE /api/social/follows/{id}` |
| GET | `/social/following/{id}` | 公开 | 关注列表 | `GET /api/social/following/{id}` |
| GET | `/social/followers/{id}` | 公开 | 粉丝列表 | `GET /api/social/followers/{id}` |
| GET | `/friends` | 用户 | 好友列表 | `GET /api/friends` |
| POST | `/friends/requests` | 用户 | 发送好友申请 | `POST /api/friends/requests` |
| GET | `/friends/requests` | 用户 | 好友申请列表 | `GET /api/friends/requests` |
| POST | `/friends/requests/{id}/accept` | 接收方 | 接受好友申请 | `POST /api/friends/requests/{id}/accept` |
| POST | `/friends/requests/{id}/reject` | 接收方/发起方 | 拒绝或撤销好友申请 | `POST /api/friends/requests/{id}/reject` |
| DELETE | `/friends/{id}` | 用户 | 删除好友 | `DELETE /api/friends/{id}` |

## 12. 私信接口

| 方法 | 路径 | 权限 | 说明 | 旧接口 |
| --- | --- | --- | --- | --- |
| GET | `/messages/conversations` | 用户 | 私信会话列表 | `GET /api/messages/conversations` |
| POST | `/messages/conversations` | 用户 | 创建或打开私信会话 | `POST /api/messages/conversations` |
| GET | `/messages/conversations/{id}` | 会话参与者 | 会话详情 | `GET /api/messages/conversations/{id}` |
| GET | `/messages/conversations/{id}/messages` | 会话参与者 | 消息列表 | `GET /api/messages/conversations/{id}/messages` |
| POST | `/messages/conversations/{id}/messages` | 会话参与者 | 发送消息 | `POST /api/messages/conversations/{id}/messages` |

创建会话请求：

```json
{
  "target_user_id": "bob"
}
```

发送消息请求：

```json
{
  "content": "你好"
}
```

## 13. 群聊接口

| 方法 | 路径 | 权限 | 说明 | 旧接口 |
| --- | --- | --- | --- | --- |
| GET | `/groups` | 公开 | 群聊列表 | `GET /api/groups` |
| POST | `/groups` | 用户 | 创建群聊 | `POST /api/groups` |
| GET | `/groups/{id}` | 公开 | 群聊详情 | `GET /api/groups/{id}` |
| PUT | `/groups/{id}` | 群主/管理员 | 修改群资料 | `PUT /api/groups/{id}` |
| DELETE | `/groups/{id}` | 群主/管理员 | 解散群聊 | `DELETE /api/groups/{id}` |
| GET | `/groups/{id}/members` | 群成员 | 成员列表 | `GET /api/groups/{id}/members` |
| POST | `/groups/{id}/members` | 用户/管理员 | 加入或邀请成员 | `POST /api/groups/{id}/members` |
| DELETE | `/groups/{id}/members/{user_id}` | 群主/管理员 | 移除成员 | `DELETE /api/groups/{id}/members/{user_id}` |
| GET | `/groups/{id}/messages` | 群成员 | 群消息列表 | `GET /api/groups/{id}/messages` |
| POST | `/groups/{id}/messages` | 群成员 | 发送群消息 | `POST /api/groups/{id}/messages` |

创建群聊请求：

```json
{
  "name": "Web 安全交流",
  "intro": "Web 方向讨论",
  "avatar_url": "/uploads/groups/web.png",
  "visibility": "public",
  "member_user_ids": ["alice", "bob"]
}
```

## 14. 通知与搜索接口

| 方法 | 路径 | 权限 | 说明 | 旧接口 |
| --- | --- | --- | --- | --- |
| GET | `/notifications` | 用户 | 通知列表 | `GET /api/notifications` |
| POST | `/notifications/{id}/read` | 用户 | 标记已读 | `POST /api/notifications/{id}/read` |
| GET | `/search` | 公开 | 全站搜索 | `GET /api/search` |

搜索参数：

```text
q=关键词
type=all|article|user|tag
page=1
page_size=20
```

## 15. 最新消息模块

新增模块。

### 15.1 公开读取

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| GET | `/announcements` | 公开 | 最新消息列表 |
| GET | `/announcements/{id}` | 公开 | 最新消息详情 |

### 15.2 后台管理

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| GET | `/admin/announcements` | 管理员 | 后台消息列表 |
| POST | `/admin/announcements` | 管理员 | 新增消息 |
| PUT | `/admin/announcements/{id}` | 管理员 | 修改消息 |
| DELETE | `/admin/announcements/{id}` | 管理员 | 删除消息 |
| PUT | `/admin/announcements/{id}/visibility` | 管理员 | 启用 / 隐藏消息 |

请求示例：

```json
{
  "title": "招新宣讲通知",
  "summary": "本周五晚开展招新宣讲",
  "content": "详细内容...",
  "category": "招新",
  "cover_url": "/uploads/news/cover.jpg",
  "pinned": true,
  "visible": true,
  "published_at": "2026-05-11T20:00:00+08:00"
}
```

## 16. 常用工具模块

新增模块。

### 16.1 公开读取

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| GET | `/tools/categories` | 公开 | 工具分类列表 |
| GET | `/tools` | 公开 | 工具列表 |
| GET | `/tools/{id}` | 公开 | 工具详情 |
| POST | `/tools/{id}/click` | 公开 | 记录点击 |

### 16.2 后台管理

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| GET | `/admin/tools/categories` | 管理员 | 工具分类管理列表 |
| POST | `/admin/tools/categories` | 管理员 | 新增工具分类 |
| PUT | `/admin/tools/categories/{id}` | 管理员 | 修改工具分类 |
| DELETE | `/admin/tools/categories/{id}` | 管理员 | 删除工具分类 |
| GET | `/admin/tools` | 管理员 | 工具管理列表 |
| POST | `/admin/tools` | 管理员 | 新增工具 |
| PUT | `/admin/tools/{id}` | 管理员 | 修改工具 |
| DELETE | `/admin/tools/{id}` | 管理员 | 删除工具 |

工具请求示例：

```json
{
  "category_id": "cat_xxx",
  "name": "RAG 问答",
  "description": "团队知识库问答工具",
  "icon_url": "/uploads/icons/rag.png",
  "url": "/rag",
  "sort_order": 10,
  "enabled": true
}
```

## 17. 常用网站导航模块

新增模块。

### 17.1 公开读取

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| GET | `/navigation/categories` | 公开 | 网站分类列表 |
| GET | `/navigation/sites` | 公开 | 网站导航列表 |
| GET | `/navigation/sites/{id}` | 公开 | 网站详情 |
| POST | `/navigation/sites/{id}/click` | 公开 | 记录点击 |

### 17.2 后台管理

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| GET | `/admin/navigation/categories` | 管理员 | 导航分类管理列表 |
| POST | `/admin/navigation/categories` | 管理员 | 新增导航分类 |
| PUT | `/admin/navigation/categories/{id}` | 管理员 | 修改导航分类 |
| DELETE | `/admin/navigation/categories/{id}` | 管理员 | 删除导航分类 |
| GET | `/admin/navigation/sites` | 管理员 | 网站导航管理列表 |
| POST | `/admin/navigation/sites` | 管理员 | 新增网站 |
| PUT | `/admin/navigation/sites/{id}` | 管理员 | 修改网站 |
| DELETE | `/admin/navigation/sites/{id}` | 管理员 | 删除网站 |

网站请求示例：

```json
{
  "category_id": "cat_xxx",
  "name": "GitHub",
  "url": "https://github.com",
  "description": "代码托管平台",
  "icon_url": "/uploads/icons/github.png",
  "recommended": true,
  "sort_order": 10,
  "enabled": true
}
```

## 18. 一键关站接口

新增模块。

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| GET | `/maintenance/state` | 公开 | 获取当前维护状态 |
| GET | `/admin/maintenance/state` | 管理员 | 后台获取维护状态 |
| PUT | `/admin/maintenance/state` | 超级管理员/站点管理员 | 更新维护状态 |
| POST | `/admin/maintenance/close` | 超级管理员/站点管理员 | 一键关站 |
| POST | `/admin/maintenance/open` | 超级管理员/站点管理员 | 恢复开站 |

关站请求：

```json
{
  "notice": "网站维护中，请稍后再试。",
  "reason": "系统升级",
  "expected_restore_at": "2026-05-11T10:00:00+08:00"
}
```

维护状态响应：

```json
{
  "success": true,
  "data": {
    "closed": true,
    "notice": "网站维护中，请稍后再试。",
    "reason": "系统升级",
    "expected_restore_at": "2026-05-11T10:00:00+08:00",
    "updated_by": "admin",
    "updated_at": "2026-05-10T21:00:00+08:00"
  }
}
```

## 19. 后台社区管理与审计接口

### 19.1 博客站状态

| 方法 | 路径 | 权限 | 说明 | 旧接口 |
| --- | --- | --- | --- | --- |
| GET | `/admin/blog/site-state` | 管理员 | 获取博客站开放状态 | `GET /api/admin/blog/site-state` |
| PUT | `/admin/blog/site-state` | 管理员 | 修改博客站开放状态 | `PUT /api/admin/blog/site-state` |

### 19.2 社区用户管理

| 方法 | 路径 | 权限 | 说明 | 旧接口 |
| --- | --- | --- | --- | --- |
| GET | `/admin/community/users` | 管理员 | 社区用户列表 | `GET /api/admin/community/users` |
| POST | `/admin/community/users` | 管理员 | 新增社区用户 | `POST /api/admin/community/users` |
| PUT | `/admin/community/users/{id}` | 管理员 | 修改社区用户 | `PUT /api/admin/community/users/{id}` |
| DELETE | `/admin/community/users/{id}` | 管理员 | 删除社区用户 | `DELETE /api/admin/community/users/{id}` |
| PUT | `/admin/community/users/{id}/status` | 管理员 | 修改用户状态 | `PUT /api/admin/community/users/{id}/status` |
| PUT | `/admin/community/users/{id}/role` | 管理员 | 修改用户角色 | `PUT /api/admin/community/users/{id}/role` |
| PUT | `/admin/community/users/{id}/password` | 管理员 | 重置用户密码 | `PUT /api/admin/community/users/{id}/password` |

### 19.3 超级管理员审计

| 方法 | 路径 | 权限 | 说明 | 旧接口 |
| --- | --- | --- | --- | --- |
| GET | `/superadmin/audit/private-conversations` | 超级管理员 | 私信会话审计 | `GET /api/superadmin/audit/private-conversations` |
| GET | `/superadmin/audit/private-conversations/{id}/messages` | 超级管理员 | 私信消息审计 | `GET /api/superadmin/audit/private-conversations/{id}/messages` |
| GET | `/superadmin/audit/groups` | 超级管理员 | 群聊审计 | `GET /api/superadmin/audit/groups` |
| GET | `/superadmin/audit/groups/{id}/messages` | 超级管理员 | 群消息审计 | `GET /api/superadmin/audit/groups/{id}/messages` |
| GET | `/admin/audit-logs` | 管理员 | 管理后台操作日志 | 新增 |

## 20. 旧接口迁移原则

旧接口迁移到新接口时遵循：

- 旧 `/api/*` 不再作为新前端依赖。
- 新前端只使用 `/api/v1/*`。
- 迁移期间可临时保留旧接口，内部转发到新 service。
- 旧接口下线前需要完成前端替换和测试确认。

典型映射：

| 旧接口 | 新接口 |
| --- | --- |
| `/api/content` | `/api/v1/site/content` |
| `/api/news/{id}` | `/api/v1/news/{id}` |
| `/api/recruit/apply` | `/api/v1/recruit/applications` |
| `/api/admin/login` | `/api/v1/admin/auth/login` |
| `/api/users/login` | `/api/v1/auth/login` |
| `/api/blog/articles` | `/api/v1/blog/articles` |
| `/api/chat` | `/api/v1/rag/chat` |
| `/api/upload` | `/api/v1/uploads/pdf` |

## 21. 前端 API 模块对应关系

```text
frontend/src/api/auth.ts           -> /auth, /users/me
frontend/src/api/admin.ts          -> /admin/*
frontend/src/api/site.ts           -> /site, /news, /awards, /seniors, /reviews
frontend/src/api/recruit.ts        -> /recruit/*
frontend/src/api/upload.ts         -> /uploads/*
frontend/src/api/rag.ts            -> /rag/*
frontend/src/api/user.ts           -> /users/*
frontend/src/api/blog.ts           -> /blog/*
frontend/src/api/social.ts         -> /social, /friends
frontend/src/api/message.ts        -> /messages/*
frontend/src/api/group.ts          -> /groups/*
frontend/src/api/notification.ts   -> /notifications
frontend/src/api/announcement.ts   -> /announcements
frontend/src/api/tools.ts          -> /tools
frontend/src/api/navigation.ts     -> /navigation
frontend/src/api/maintenance.ts    -> /maintenance, /admin/maintenance
```

## 22. 后续补充内容

本文档当前定义 API 分组、路径、权限和核心请求结构。后续实现时还应补充：

- 每个接口的完整请求字段约束
- 每个接口的完整响应字段
- 错误码枚举
- OpenAPI / Swagger 文档
- 接口测试用例
- 前后端联调样例

