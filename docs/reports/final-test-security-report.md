# Flyteam Website 最终功能测试与安全测试报告

> 报告日期：2026-05-10  
> 项目目录：`E:\学校\Flyteam\rag`  
> 本地测试地址：`http://127.0.0.1:8000`  
> 测试对象：宣传站 + 后台管理 + 博客社区 + 聊天系统 + RAG 问答 + 上传与安全防护  
> 说明：报告不记录任何明文密码、API Key 或 VPS 登录凭据。

---

## 1. 总体结论

本轮检查完成后，项目在本地环境中已通过核心功能测试、权限测试、博客站开放/关闭开关测试、接口冒烟测试、构建测试和基础安全测试。

### 1.1 结论摘要

| 项目 | 结果 |
|---|---:|
| Go 自动化测试 | 通过 |
| Go vet 静态检查 | 通过 |
| Go build 构建 | 通过 |
| 前端 JS 语法检查 | 通过 |
| HTTP 冒烟测试 | 48 / 48 通过 |
| 安全响应头检查 | 7 / 7 通过 |
| 未登录访问后台保护 | 通过 |
| CSRF 防护测试 | 通过 |
| 路径穿越测试 | 通过 |
| 静态备份文件泄露防护 | 已修复并通过 |
| 博客站关闭开关 | 通过 |
| 敏感信息入库/入 Git 检查 | 未发现明文密码、VPS IP 泄露；`.env` 未被 Git 跟踪 |

### 1.2 当前服务状态

- 本地服务已经重启并运行在：`http://127.0.0.1:8000`
- `/api/status` 返回：`ready=true, chunks=0`
- RAG 当前知识块数量为 `0`，因此问答接口可用，但会返回“未检索到资料”的兜底回答。
- 博客站当前已恢复为开放状态。

---

## 2. 本轮新增/修复内容

### 2.1 超级管理员控制博客站开放状态

新增能力：超级管理员可以在后台控制博客站是否对外开放。

涉及文件：

| 文件 | 说明 |
|---|---|
| `cmd/flyteam-server/internal/app/admin_blog_site_state.go` | 新增博客站开放状态存储、读取、更新、关闭页渲染、博客 API 拦截判断 |
| `cmd/flyteam-server/internal/app/admin_community_audit.go` | 新增 `/api/admin/blog/site-state` 管理接口路由 |
| `cmd/flyteam-server/main.go` | 在请求入口处接入博客页面/API 关闭拦截 |
| `app/static/admin.html` | 后台新增“博客开关”面板 |
| `app/static/app.js` | 后台新增状态读取、保存、权限控制逻辑 |
| `cmd/flyteam-server/internal/app/admin_community_audit_test.go` | 新增博客开关权限和访问拦截测试 |

设计结果：

- 默认状态：博客站开放。
- 超级管理员可关闭/开启。
- 博客管理员可读取状态，但不能修改状态。
- 宣传站管理员不能读取/修改该状态。
- 关闭后，普通用户无法访问博客页面、登录注册、空间、私信、群聊和博客相关 API。
- 关闭只是访问层拦截，不删除数据库、文章、聊天记录、上传文件或缓存。
- 博客管理员/超级管理员仍可绕过关闭状态进行后台检查。

### 2.2 静态备份文件泄露防护

测试时发现：`/static/app.js.codex_backup` 能被直接访问，属于静态备份文件暴露风险。

已修复：

- 在 `serveFileRoot` 增加 `blockedPublicFile` 检查。
- 阻止以下类型文件被 `/static/` 或 `/uploads/` 直接访问：
  - 点文件：`.env`、`.git` 类路径
  - 备份文件：`.bak`、`.backup`、`.old`、`.orig`、`.codex_backup`
  - 临时文件：`.tmp`、`.temp`、`.swp`
  - 日志/数据库：`.log`、`.db`、`.sqlite`、`.sqlite3`
  - 服务端源码/脚本：`.go`、`.py`、`.ps1`、`.sh`、`.bat`、`.cmd`
- 新增自动化测试 `TestStaticBackupAndSecretFilesAreNotServed`。
- 本地验证：`/static/app.js.codex_backup` 现在返回 `404`。

---

## 3. 当前系统功能清单

### 3.1 宣传站前台

| 模块 | 功能 |
|---|---|
| 首页 | 全屏照片墙、随机/流动背景、导航跳转、简洁展示 |
| 团队新闻 | 新闻列表、详情、图片展示、富文本/标题排版、置顶、排序 |
| 团队回顾 | 栏目型相册，支持摘要、正文、多图、详情页 |
| 奖项荣誉 | 团队赛/个人赛，国家级/省级分类，国家级优先，同级按时间排序，置顶 |
| Flyteamers/前辈墙 | 年级、帮主、团队管理、负责人标记、照片展示、双击放大 |
| 招新报名 | 动态 C 语言验证码、提交报名、美化按钮、后台查看 |
| 团队简介 | 后台维护文案，前台展示 |
| RAG 助手 | 基于 PDF 知识库问答，当前本地状态为 ready，但 chunks=0 |

### 3.2 宣传站后台

| 模块 | 功能 |
|---|---|
| 后台登录 | 管理员登录、Session、自动超时 |
| 首页轮播管理 | 图片上传、删除、首页随机展示 |
| 新闻管理 | 新增、编辑、删除、上传图片、置顶、富文本排版 |
| 回顾管理 | 新增栏目、编辑栏目、管理多图、置顶、详情页 |
| 奖项管理 | 新增/编辑/删除，团队赛/个人赛，国家级/省级，置顶 |
| 前辈墙管理 | 新增/编辑/删除，负责人、帮主、团队管理等标记，上传图片 |
| 报名管理 | 查看、搜索、编辑、删除报名记录 |
| 知识库管理 | 上传 PDF、重建默认知识库、RAG 配置状态 |

### 3.3 博客社区

| 模块 | 功能 |
|---|---|
| 普通用户注册 | 昵称、ID、密码注册，注册后默认待审核 |
| 注册审核 | 博客管理员/超级管理员批准后才能登录，驳回后释放 ID |
| 博客广场 | 推荐文章、热门文章、搜索、标签/分类展示 |
| 文章系统 | 发布、编辑、标题、图片、代码/文本格式、浏览量、推荐排序 |
| 个人空间 | 用户主页、文章列表、头像、资料展示 |
| 个人中心 | 修改头像、昵称、账号信息等 |
| 评论/互动 | 登录用户评论、点赞、收藏等 |
| 关注/好友 | 类似申请好友逻辑，关注/好友管理 |
| 私信 | 好友私聊，聊天记录入库 |
| 群聊 | 建群、拉人、群消息，聊天记录入库 |
| 博客站开关 | 超级管理员可临时关闭博客站对外访问，缓存不变 |

### 3.4 权限体系

| 权限 | 能力 |
|---|---|
| 普通用户 | 使用博客站、看文章、评论、互动、私聊/群聊 |
| 宣传站管理员 `site_admin` | 管理宣传站内容、报名、知识库、轮播等 |
| 博客站管理员 `blog_admin` | 审核注册用户、管理博客用户、处理博客用户状态 |
| 超级管理员 `superadmin` | 全站唯一；管理两个后台管理员；审计聊天记录；控制博客开放状态 |

---

## 4. 自动化测试结果

### 4.1 执行命令

```powershell
Set-Location -LiteralPath 'E:\学校\Flyteam\rag'
node --check app\static\app.js
go test ./...
go test -cover ./...
go vet ./...
go build ./cmd/flyteam-server
```

### 4.2 结果

| 命令 | 结果 |
|---|---|
| `node --check app/static/app.js` | 通过 |
| `go test ./...` | 通过 |
| `go test -cover ./...` | 通过，覆盖率 24.1% |
| `go vet ./...` | 通过 |
| `go build ./cmd/flyteam-server` | 通过 |

### 4.3 当前 Go 测试用例清单

| 测试用例 | 覆盖内容 |
|---|---|
| `TestAdminCommunityUsersAndSuperAuditPermissions` | 博客管理员、宣传站管理员、超级管理员权限分层；审计接口权限 |
| `TestAdminRoleSplitAllowsOnlyOneSuperAdmin` | 超级管理员唯一性，禁止创建第二个超级管理员，禁止降级唯一超级管理员 |
| `TestCommunityRegistrationRequiresAdminApproval` | 新用户注册待审核、待审核禁止登录、博客管理员批准后可登录 |
| `TestCommunityRegistrationRejectReleasesUserID` | 注册驳回后删除待审核记录，释放用户 ID |
| `TestBlogSiteOpenStatePermissionsAndAccessGate` | 博客开关权限、关闭后阻断普通访问、管理员可检查、关闭不删除数据 |
| `TestStaticBackupAndSecretFilesAreNotServed` | 静态备份/点文件/数据库/脚本文件禁止直接访问 |
| `TestDLCommentsReactionsAndNotifications` | 评论、点赞、收藏、通知、计数器 |
| `TestDLFollowMessagesAndGroups` | 关注/好友、私信、群聊、成员权限 |
| `TestDLSearchAndRecommendations` | 博客搜索与推荐排序 |

---

## 5. HTTP 冒烟测试结果

本轮对本地服务执行 48 项接口/页面/安全冒烟测试，全部通过。

### 5.1 公共页面

| 页面 | 期望 | 实际 | 结果 |
|---|---:|---:|---|
| `/` | 200 | 200 | 通过 |
| `/flyteamers` | 200 | 200 | 通过 |
| `/recruit` | 200 | 200 | 通过 |
| `/news` | 200 | 200 | 通过 |
| `/awards` | 200 | 200 | 通过 |
| `/review` | 200 | 200 | 通过 |
| `/intro` | 200 | 200 | 通过 |
| `/blog` | 200 | 200 | 通过 |
| `/user-login` | 200 | 200 | 通过 |
| `/user-register` | 200 | 200 | 通过 |
| `/messages` | 200 | 200 | 通过 |
| `/groups` | 200 | 200 | 通过 |
| `/space/smoke` | 200 | 200 | 通过 |

### 5.2 公共接口

| 接口 | 期望 | 实际 | 结果 |
|---|---:|---:|---|
| `/api/status` | 200 | 200 | 通过 |
| `/api/content` | 200 | 200 | 通过 |
| `/api/recruit/captcha` | 200 | 200 | 通过 |
| `/api/recruit/halls` | 200 | 200 | 通过 |
| `/api/recruit/stats` | 200 | 200 | 通过 |
| `/api/blog/recommendations` | 200 | 200 | 通过 |
| `/api/chat` 空知识库兜底 | 200 | 200 | 通过 |

RAG 空知识库返回：

```json
{"answer":"未检索到与问题相关的资料，当前无法回答该问题。","sources":[]}
```

---

## 6. 安全测试结果

### 6.1 安全响应头

| 响应头 | 当前值/状态 | 结果 |
|---|---|---|
| `X-Content-Type-Options` | `nosniff` | 通过 |
| `X-Frame-Options` | `SAMEORIGIN` | 通过 |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | 通过 |
| `Permissions-Policy` | 禁用 geolocation/microphone/camera | 通过 |
| `Content-Security-Policy` | 已配置 default-src/self、frame-ancestors、自身连接等 | 通过 |
| `Cross-Origin-Opener-Policy` | `same-origin` | 通过 |
| `Cross-Origin-Resource-Policy` | `same-origin` | 通过 |

备注：CSP 当前为了兼容 Vue Runtime 模板包含 `unsafe-eval`，可运行，但不是最强安全配置。后续如果把后台前端改成预编译或不依赖运行时模板，可以去掉 `unsafe-eval`。

### 6.2 后台鉴权

| 测试项 | 期望 | 实际 | 结果 |
|---|---:|---:|---|
| 未登录访问 `/admin` | 302/303 跳转登录 | 302 | 通过 |
| 未登录访问 `/static/admin.html` | 302/303 跳转登录 | 302 | 通过 |
| 未登录读取 `/api/recruit/list` | 401 | 401 | 通过 |
| 未登录新增新闻 `/api/news` | 401 | 401 | 通过 |
| 未登录上传新闻图片 | 401 | 401 | 通过 |

### 6.3 CSRF 防护

测试：使用 Cookie 会话但不携带 CSRF Token 调用后台修改接口。

| 测试项 | 期望 | 实际 | 结果 |
|---|---:|---:|---|
| Cookie 会话无 CSRF 修改团队概况 | 403 | 403 | 通过 |

结论：后台 Cookie 会话的变更类接口具备 CSRF 校验。

### 6.4 博客站关闭开关安全测试

| 测试项 | 期望 | 实际 | 结果 |
|---|---|---|---|
| 超级管理员读取博客站状态 | 成功 | 成功 | 通过 |
| 超级管理员关闭博客站 | `open=false` | `open=false` | 通过 |
| 关闭后普通访问 `/blog` | 503 | 503 | 通过 |
| 关闭后普通访问 `/api/blog/recommendations` | 503 | 503 | 通过 |
| 关闭后超级管理员检查博客 API | 200 | 200 | 通过 |
| 恢复博客站原始状态 | 成功 | 成功 | 通过 |

结论：博客站开关是访问控制，不会清空数据。测试中关闭后再恢复，状态已恢复开放。

### 6.5 路径穿越与静态文件泄露

| 测试项 | 期望 | 实际 | 结果 |
|---|---:|---:|---|
| `/static/%2e%2e/.env` | 404/400 | 404 | 通过 |
| `/uploads/%2e%2e/storage/flyteam.db` | 404/400 | 404 | 通过 |
| `/static/..%2f.env` | 404/400 | 404 | 通过 |
| `/static/app.js.codex_backup` | 404/400 | 404 | 通过 |

结论：路径穿越与静态备份文件泄露当前已被拦截。

### 6.6 文件上传安全

代码检查结果：

| 防护点 | 当前实现 |
|---|---|
| 后台上传鉴权 | 上传接口需要管理员权限 |
| 博客图片上传鉴权 | 需要社区用户登录 |
| 文件后缀白名单 | 图片仅允许 jpg/jpeg/png/webp/gif；PDF 仅允许 pdf |
| 文件魔术头校验 | 图片检测 JPEG/PNG/GIF/WebP；PDF 检测 `%PDF-` |
| 扩展名和实际格式匹配 | 已检查 |
| 危险内容检测 | 阻断 `<?php`、`<script`、JSP/ASP 片段、shebang 等 |
| PDF 主动内容检测 | 阻断 `/JavaScript`、`/OpenAction`、`/Launch`、`/EmbeddedFile` 等 |
| 单文件大小限制 | 已通过 `LimitReader` 限制 |
| 文件数量限制 | 已限制最大上传数量 |
| 文件名安全 | 使用随机文件名，不使用原始文件名落盘 |
| 路径安全 | 保存目录固定，静态读取会校验路径在根目录内 |

### 6.7 SQL 注入风险检查

检查结果：

- 主体 SQL 均使用 `?` 参数绑定。
- 发现一处动态拼接表名逻辑，用于点赞/收藏表切换。
- 该表名不是来自用户输入，而是服务端根据固定 `kind` 分支决定：`blog_likes` 或 `blog_favorites`。
- 当前不构成可控 SQL 注入。

### 6.8 XSS/前端注入风险检查

检查结果：

- 前端有多处 `innerHTML`，但核心博客/聊天渲染中已使用 `escapeHTML` 进行转义。
- 聊天消息支持换行显示，使用先转义再替换换行的方式。
- Vue 模板默认插值会转义 HTML。
- 未发现 `eval()`、`document.write()`、`v-html` 直接渲染不可信数据。

剩余建议：

- 后续新增页面时必须继续遵循“用户内容先转义再插入 DOM”。
- 如果将来加入 Markdown HTML 直出，必须接入白名单 HTML Sanitizer。

### 6.9 敏感信息检查

| 检查项 | 结果 |
|---|---|
| `.env` 是否被 Git 跟踪 | 未跟踪 |
| 超级管理员密码明文是否在 Git 仓库中 | 未发现 |
| VPS IP 是否在 Git 仓库中 | 未发现 |
| `DASHSCOPE_API_KEY` / `OPENAI_API_KEY` 关键词 | 仅为 `.env.example`、README 占位符和代码中的环境变量名；未发现真实 Key |

---

## 7. 已确认的核心业务功能测试覆盖

### 7.1 招新报名

| 功能点 | 结果 |
|---|---|
| 报名页可访问 | 通过 |
| 堂口接口可访问 | 通过 |
| 报名统计接口可访问 | 通过 |
| C 语言动态验证码接口可访问 | 通过 |
| 后台报名列表未登录不可访问 | 通过 |
| 后台报名管理需要宣传站管理员权限 | 已由权限逻辑覆盖 |

### 7.2 新闻/回顾/奖项/前辈墙

| 功能点 | 结果 |
|---|---|
| 前台页面可访问 | 通过 |
| 后台新增/编辑/删除需要宣传站管理员权限 | 通过鉴权逻辑和代码检查 |
| 置顶逻辑 | 已接入各模块数据结构与后台控制 |
| 排序逻辑 | 新闻/回顾按登记/时间排序，奖项国家级优先、省级靠后，同级按时间排序 |
| 图片双击放大 | 前端已接入展示图片交互 |

### 7.3 博客社区

| 功能点 | 结果 |
|---|---|
| 博客首页可访问 | 通过 |
| 登录页可访问 | 通过 |
| 注册页可访问 | 通过 |
| 新注册默认待审核 | 自动化测试通过 |
| 待审核用户不可登录 | 自动化测试通过 |
| 博客管理员/超级管理员可审核 | 自动化测试通过 |
| 驳回释放用户 ID | 自动化测试通过 |
| 评论/点赞/收藏/通知 | 自动化测试通过 |
| 关注/好友 | 自动化测试通过 |
| 私信 | 自动化测试通过 |
| 群聊 | 自动化测试通过 |
| 搜索和推荐 | 自动化测试通过 |

### 7.4 管理员体系

| 功能点 | 结果 |
|---|---|
| 宣传站管理员和博客站管理员分离 | 自动化测试通过 |
| 博客管理员不能管理宣传站报名 | 自动化测试通过 |
| 宣传站管理员不能管理博客用户 | 自动化测试通过 |
| 超级管理员全站唯一 | 自动化测试通过 |
| 禁止创建第二个超级管理员 | 自动化测试通过 |
| 禁止降级唯一超级管理员 | 自动化测试通过 |
| 超级管理员可审计私信/群聊 | 自动化测试通过 |
| 普通管理员不可审计聊天记录 | 自动化测试通过 |

---

## 8. 当前未完全覆盖/建议后续增强项

这些不是当前阻断问题，但建议后续完善。

### 8.1 测试覆盖率仍偏低

当前 Go 测试覆盖率：`24.1%`。

建议后续增加：

- 内容管理 CRUD 更完整的自动化测试。
- 上传真实图片/PDF 的自动化测试。
- 招新报名完整提交流程测试。
- RAG 入库、重建、问答召回测试。
- 浏览器端 E2E 测试，例如 Playwright。

### 8.2 Race 检测未在本机完成

尝试执行：

```powershell
go test -race ./cmd/flyteam-server
```

结果：当前 Windows 本机缺少 C 编译器 `gcc`，Go race 需要 CGO，因此本机无法完成 race 检测。

建议：

- 在 GitHub Actions Ubuntu 环境中运行：

```bash
go test -race ./cmd/flyteam-server
```

### 8.3 CSP 仍包含 `unsafe-eval`

原因：当前后台依赖 Vue Runtime 模板解析。

风险：如果未来存在 XSS，`unsafe-eval` 会降低 CSP 防御强度。

建议：

- 后续将 Vue 页面预编译，或改为不依赖 runtime template。
- 然后去掉 `script-src` 中的 `unsafe-eval`。

### 8.4 Token 仍会进入 localStorage

当前系统同时支持：

- HttpOnly Cookie 会话。
- 前端 Header Token 模式，部分 token 会存入 localStorage。

风险：如果未来出现 XSS，localStorage 中的 token 可能被读取。

建议增强：

- 管理员后台尽量迁移到 Cookie-only + CSRF 模式。
- 普通用户也尽量减少 localStorage token 的使用。
- 保留 CSRF Token 单独机制。

### 8.5 RAG 当前为空知识库

当前 `/api/status` 显示 `chunks=0`。

结果：问答功能可用，但没有资料时只能返回兜底回答。

部署/使用建议：

- 在 VPS 的 `.env` 中配置 `DASHSCOPE_API_KEY` 或 `OPENAI_API_KEY`。
- 后台上传指定 PDF。
- 点击“重建默认知识库”。
- 再检查 `/api/status` 的 chunks 数量是否大于 0。

### 8.6 第三方依赖漏洞扫描未做在线校验

由于当前环境没有额外联网漏洞库扫描，本轮未执行 `govulncheck` 在线漏洞库校验。

建议后续在 CI 加：

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

---

## 9. 部署安全建议

### 9.1 VPS 环境变量

生产环境 `.env` 建议：

```env
PORT=8000
ADMIN_COOKIE_SECURE=true
DASHSCOPE_API_KEY=你的真实Key
# 或 OPENAI_API_KEY=你的真实Key
```

不要提交 `.env` 到 Git。

### 9.2 Nginx / HTTPS

建议：

- 域名走 HTTPS。
- Nginx 反代到本地 Go 服务端口。
- 开启 HTTP -> HTTPS 重定向。
- 上传目录不要允许执行脚本。
- 不要把项目根目录直接交给 Nginx 静态托管。

### 9.3 数据库和上传文件

建议：

- SQLite 数据库定期备份。
- `storage/`、`uploads/` 不要提交 Git。
- 上传目录只保留读写权限，不给执行权限。
- 定期清理无用备份文件。

### 9.4 Git 协作

建议：

- 主分支开启 PR 审核。
- 禁止直接推送 main。
- CI 至少跑：

```bash
go test ./...
go vet ./...
go build ./cmd/flyteam-server
node --check app/static/app.js
```

后续增强：

```bash
go test -race ./cmd/flyteam-server
govulncheck ./...
```

---

## 10. 最终结论

当前版本已经满足：

1. 宣传站已有功能不丢失。
2. 博客站基础功能、注册审核、聊天、群聊、关注、文章推荐等核心链路存在并通过测试。
3. 宣传站管理员、博客站管理员、超级管理员权限已拆分。
4. 超级管理员可以控制博客站是否对外开放。
5. 关闭博客站不会删除任何文章、用户、聊天记录、图片或缓存。
6. 后台和敏感接口已有鉴权和 CSRF 防护。
7. 上传文件有后缀、魔术头、危险内容、大小和数量限制。
8. 路径穿越和静态备份文件泄露已做防护。
9. 本地构建与测试通过。

当前建议你重点人工复核的只有两点：

- 后台“博客开关”界面的实际视觉是否满意。
- RAG 上传 PDF 后的真实问答质量，因为当前本地知识库 chunks=0，只能测兜底逻辑。
