# CI/CD 部署说明

本项目已经配置 GitHub Actions：

- Pull Request / push 到 `main`、`develop`：自动执行 CI。
- push 到 `main`：CI 通过后，如果 VPS Secrets 已配置，则自动部署到 VPS。
- Actions 页面手动执行 `CI/CD` workflow：可手动触发部署。

## 1. CI 会检查什么

工作流文件：

```text
.github/workflows/ci-cd.yml
```

CI 步骤：

1. Checkout 代码。
2. 安装 Go 1.22.x。
3. 安装 Node.js 20。
4. `go mod download`。
5. `gofmt` 检查。
6. `go test ./...`。
7. `go vet ./...`。
8. 对 `app/static/js/*.js` 执行 `node --check`。
9. 构建 Linux amd64 二进制文件。
10. 打包部署产物并上传 Artifact。

## 2. CD 部署逻辑

CD 只上传和覆盖代码/程序文件：

```text
flyteam-server
app/
.env.example
README.md
docs/PROJECT_STRUCTURE.md
docs/DIRECTORY_MAP.md
```

不会删除或覆盖 VPS 上的：

```text
.env
storage/
storage/uploads/
storage/flyteam.db
storage/rag_index_go.json
```

部署完成后会执行：

```bash
systemctl restart flyteam-rag.service
systemctl is-active --quiet flyteam-rag.service
```

## 3. GitHub Secrets 配置

进入仓库：

```text
Settings -> Secrets and variables -> Actions -> New repository secret
```

需要添加：

| Secret 名称 | 示例值 | 说明 |
| --- | --- | --- |
| `VPS_HOST` | `154.94.237.213` | VPS IP 或域名 |
| `VPS_USER` | `root` | SSH 用户 |
| `VPS_PORT` | `43725` | SSH 端口 |
| `VPS_SSH_KEY` | 私钥全文 | GitHub Actions 用于登录 VPS 的 SSH 私钥 |
| `VPS_DEPLOY_PATH` | `/opt/flyteam-rag` | VPS 项目目录 |
| `VPS_SERVICE` | `flyteam-rag.service` | systemd 服务名 |

> 不建议在 GitHub Actions 里使用 SSH 密码。请使用专门的部署 SSH Key。

## 4. 生成部署 SSH Key

在你自己的电脑执行：

```bash
ssh-keygen -t ed25519 -C "flyteam-github-actions" -f flyteam_github_actions
```

会生成：

```text
flyteam_github_actions      # 私钥，填入 GitHub Secret: VPS_SSH_KEY
flyteam_github_actions.pub  # 公钥，放到 VPS authorized_keys
```

把公钥上传到 VPS：

```bash
ssh -p 43725 root@154.94.237.213 "mkdir -p ~/.ssh && chmod 700 ~/.ssh"
cat flyteam_github_actions.pub | ssh -p 43725 root@154.94.237.213 "cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys"
```

测试：

```bash
ssh -i flyteam_github_actions -p 43725 root@154.94.237.213 "echo ok"
```

确认成功后，把 `flyteam_github_actions` 私钥全文复制到 `VPS_SSH_KEY`。

## 5. VPS systemd 服务要求

如果 VPS 已经有 `flyteam-rag.service`，一般不用改。推荐服务内容如下：

```ini
[Unit]
Description=Flyteam Website Go Server
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/flyteam-rag
EnvironmentFile=/opt/flyteam-rag/.env
ExecStart=/opt/flyteam-rag/flyteam-server
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

保存到：

```text
/etc/systemd/system/flyteam-rag.service
```

启用：

```bash
systemctl daemon-reload
systemctl enable flyteam-rag.service
systemctl restart flyteam-rag.service
systemctl status flyteam-rag.service --no-pager
```

## 6. VPS 目录要求

VPS 项目目录建议：

```text
/opt/flyteam-rag/
├── flyteam-server
├── app/
├── .env
└── storage/
```

`.env` 需要你在 VPS 上自行维护，CI/CD 不会覆盖它。示例：

```env
PORT=8000
DATABASE_FILE=storage/flyteam.db
DASHSCOPE_API_KEY=你的真实 key
OPENAI_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
CHAT_MODEL=qwen-plus
EMBEDDING_MODEL=text-embedding-v4
ADMIN_COOKIE_SECURE=1
```

## 7. 手动触发部署

进入 GitHub：

```text
Actions -> CI/CD -> Run workflow
```

选择分支，`deploy` 选择 `true`。

## 8. 常见问题

### 8.1 CI 通过但没有部署

通常是 Secrets 未配置完整。检查 Actions 日志里的：

```text
Deployment skipped because one or more VPS_* secrets are not configured.
```

### 8.2 SSH 失败

检查：

- `VPS_HOST` 是否正确。
- `VPS_PORT` 是否正确。
- `VPS_USER` 是否有权限。
- `VPS_SSH_KEY` 是否是私钥全文，不是 `.pub` 公钥。
- VPS `~/.ssh/authorized_keys` 是否包含对应公钥。

### 8.3 部署成功但网站打不开

在 VPS 上检查：

```bash
systemctl status flyteam-rag.service --no-pager
journalctl -u flyteam-rag.service -n 100 --no-pager
curl -i http://127.0.0.1:8000/api/status
```

如果用了 Nginx，再检查：

```bash
nginx -t
systemctl status nginx --no-pager
```
