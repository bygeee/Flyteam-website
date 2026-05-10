#!/usr/bin/env python3
"""Generate docs/DIRECTORY_MAP.md from the current repository layout.

Run this after moving/adding/removing project files so the directory map stays in
sync with the codebase.
"""
from __future__ import annotations

import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
OUTPUT = ROOT / "docs" / "DIRECTORY_MAP.md"

TOP_LEVEL_DESCRIPTIONS = {
    ".github/": "GitHub 协作配置：CODEOWNERS、Issue 模板、PR 模板。",
    "app/static/assets/": "公共静态资源预留目录：图片、字体、默认背景等。",
    "app/static/css/": "前端样式文件。",
    "app/static/js/": "前端交互脚本。",
    "app/static/pages/": "HTML 页面模板，由 Go 后端路由加载。",
    "archive/legacy-python/": "旧 Python 版本备份占位目录，真实备份文件默认不提交。",
    "cmd/flyteam-server/": "Go 后端命令入口与 internal 分层代码。",
    "cmd/flyteam-server/internal/app/": "HTTP 应用层：配置、路由、鉴权适配、官网内容、社区接口、RAG 调度与上传处理。",
    "cmd/flyteam-server/internal/blog/": "博客领域层：文章模型、发布请求校验、标签规范化、公开响应结构。",
    "cmd/flyteam-server/internal/database/": "数据库层：SQLite 连接、Schema 初始化、app_kv JSON 存取。",
    "docs/": "项目文档总目录。",
    "docs/knowledge/": "本地知识库 PDF 草稿占位目录，PDF 默认不提交 Git。",
    "docs/planning/": "规划、路线图、多人成员协作任务分配。",
    "docs/reports/": "测试、安全、验收报告。",
    "scripts/": "项目维护脚本。",
    "storage/": "运行数据目录：数据库、上传文件、RAG 索引和日志；公开协作时默认不建议提交。",
}

FILE_DESCRIPTIONS = {
    ".env.example": "环境变量模板。",
    ".gitignore": "Git 忽略规则，排除密钥、运行数据、上传文件、日志和本地缓存。",
    ".github/workflows/ci-cd.yml": "GitHub Actions CI/CD：测试、构建、打包和可选 VPS 自动部署。",
    "CONTRIBUTING.md": "协作流程说明。",
    "README.md": "项目总说明和本地运行指南。",
    "go.mod": "Go 模块定义。",
    "go.sum": "Go 依赖校验锁定文件。",
    "docs/PROJECT_STRUCTURE.md": "目录结构约定和新增文件放置规范。",
    "docs/REFACTOR_ARCHITECTURE.md": "下一阶段 Go + Gin / Vue 前后端分离目标架构说明。",
    "docs/REFACTOR_REQUIREMENTS.md": "下一阶段重构与新增功能需求说明。",
    "docs/REFACTOR_TASK_PLAN.md": "下一阶段重构任务拆分、阶段计划与成员映射说明。",
    "docs/CI_CD.md": "GitHub Actions CI/CD 与 VPS 自动部署配置说明。",
    "docs/DIRECTORY_MAP.md": "自动生成的项目目录明细。",
    "docs/planning/blog-community-roadmap.md": "博客社区化改造路线图。",
    "docs/planning/team-task-allocation.md": "五人协作任务分配：z3/grand 后端，dong/dl/wang 前端。",
    "docs/reports/final-test-security-report.md": "功能与安全测试报告。",
    "scripts/update_directory_map.py": "自动刷新 docs/DIRECTORY_MAP.md。",
}

BACKEND_DESCRIPTIONS = {
    "admin_auth.go": "管理员后台鉴权、会话、角色权限和管理员账号管理。",
    "admin_blog_site_state.go": "管理员后台博客站开放/关闭状态与访问控制。",
    "admin_community_audit.go": "管理员/超级管理员的社区用户审核、权限管理和聊天审计接口。",
    "http_core.go": "HTTP 请求入口、安全响应头和全局前置校验。",
    "http_helpers.go": "HTTP/JSON、随机值、限流、时间、路径等通用辅助函数。",
    "http_static.go": "静态资源、上传资源和页面文件安全访问。",
    "public_content.go": "官网前台内容聚合、排序、奖项/前辈墙/新闻等核心逻辑。",
    "public_recruit_captcha.go": "官网前台招新报名动态 C 语言验证码。",
    "public_review_recruit.go": "官网前台团队回顾、相册、招新报名数据处理。",
    "routes.go": "顶层路由分发入口，按静态资源、公共前台、用户前台、管理员后台和 API 分组。",
    "routes_admin_backend.go": "管理员后台页面、管理员 API、后台鉴权/CSRF 权限边界。",
    "routes_public_api.go": "匿名可访问的官网前台只读 API。",
    "routes_public_frontend.go": "官网宣传站公共前台页面路由。",
    "routes_recruit.go": "招新报名公开提交与管理员审核路由。",
    "routes_site_admin_content.go": "宣传站内容管理后台 API 路由。",
    "routes_static.go": "静态文件和上传文件路由。",
    "routes_system_api.go": "RAG、文件上传和系统工具 API 路由。",
    "routes_user_api.go": "用户前台博客/社交/私信/群聊 API 路由。",
    "routes_user_frontend.go": "用户前台博客、个人中心、私信、群聊页面路由。",
    "server.go": "服务启动、配置加载和运行时依赖初始化。",
    "system_cache.go": "数据库缓存控制与辅助逻辑。",
    "system_database_adapter.go": "数据库访问适配器，调用 internal/database。",
    "system_rag.go": "RAG 知识库、PDF 文本提取、向量检索、问答调用。",
    "system_upload.go": "PDF、图片、头像等上传处理和文件安全校验。",
    "user_account.go": "用户前台注册、登录、资料与账号管理。",
    "user_avatar_upload_test.go": "用户头像上传测试。",
    "user_blog_articles.go": "用户前台博客文章发布、编辑、读取、浏览量等。",
    "user_blog_interactions.go": "用户前台博客评论、点赞、收藏等互动。",
    "user_blog_model.go": "用户前台博客领域适配器，调用 internal/blog。",
    "user_community_status.go": "用户前台社区预留/状态接口。",
    "user_friends.go": "用户前台好友申请与好友关系。",
    "user_groups.go": "用户前台群聊、群成员、群管理。",
    "user_search_notifications.go": "用户前台通知与搜索。",
    "user_session.go": "用户前台会话校验、登录态、CSRF 和用户权限辅助。",
    "user_social_messages.go": "用户前台关注、好友、私信等社交消息。",
}

BACKEND_PATH_DESCRIPTIONS = {
    "cmd/flyteam-server/main.go": "Go 命令入口，仅调用 internal/app.Run。",
    "cmd/flyteam-server/internal/blog/blog.go": "博客领域模型、文章请求校验、标签规范化和响应转换。",
    "cmd/flyteam-server/internal/database/database.go": "SQLite 连接、Schema 初始化、app_kv JSON 存取。",
}

FRONTEND_GROUPS = {
    "app/static/pages": "页面模板",
    "app/static/js": "前端脚本",
    "app/static/css": "样式文件",
}

RUNTIME_NOTES = [
    (".env", "本地/服务器真实环境变量，包含密钥，禁止提交到公开仓库。"),
    ("storage/flyteam.db", "SQLite 运行数据库，保存账号、内容、文章、聊天等数据。"),
    ("storage/uploads/", "后台上传图片、头像、PDF、博客图片等缓存。"),
    ("storage/chroma/", "旧版 Chroma 向量库缓存，如存在则不建议提交到公开仓库。"),
    ("storage/*.json", "兼容旧版 JSON 数据和迁移来源，不建议提交到公开仓库。"),
    ("storage/*.log", "运行日志，不提交。"),
    (".venv/", "本地 Python 虚拟环境，不提交。"),
    ("archive/legacy-python/*.codex_backup", "旧 Python 备份文件，不提交。"),
]


def git_tracked_files() -> list[str]:
    """Return tracked + untracked-not-ignored paths using raw UTF-8 names.

    `git ls-files` defaults to quoted octal escapes for non-ASCII paths on many
    Windows installations. `core.quotePath=false` plus `-z` keeps directory maps
    readable for uploaded Chinese filenames.
    """
    result = subprocess.run(
        ["git", "-c", "core.quotePath=false", "ls-files", "-z", "--cached", "--others", "--exclude-standard"],
        cwd=ROOT,
        check=True,
        capture_output=True,
    )
    raw = result.stdout.decode("utf-8", errors="replace")
    return sorted(p.replace("\\", "/") for p in raw.split("\0") if p)


def build_tree(paths: list[str]) -> str:
    tree: dict[str, dict] = {}
    for path in paths:
        parts = path.split("/")
        node = tree
        for part in parts:
            node = node.setdefault(part, {})

    lines: list[str] = ["."]

    def walk(node: dict[str, dict], prefix: str = "") -> None:
        items = sorted(node.items(), key=lambda kv: (bool(kv[1]), kv[0].lower()))
        for idx, (name, child) in enumerate(items):
            connector = "└── " if idx == len(items) - 1 else "├── "
            suffix = "/" if child else ""
            lines.append(f"{prefix}{connector}{name}{suffix}")
            if child:
                extension = "    " if idx == len(items) - 1 else "│   "
                walk(child, prefix + extension)

    walk(tree)
    return "\n".join(lines)


def table(rows: list[tuple[str, str]]) -> str:
    out = ["| 路径 | 说明 |", "| --- | --- |"]
    for path, desc in rows:
        out.append(f"| `{path}` | {desc} |")
    return "\n".join(out)


def collect_rows(paths: list[str]) -> tuple[list[tuple[str, str]], list[tuple[str, str]], list[tuple[str, str]]]:
    top_rows: list[tuple[str, str]] = []
    for key, desc in TOP_LEVEL_DESCRIPTIONS.items():
        normalized = key.rstrip("/")
        if any(p == normalized or p.startswith(key) for p in paths) or key == "storage/":
            top_rows.append((key, desc))

    file_rows = [(p, FILE_DESCRIPTIONS[p]) for p in sorted(FILE_DESCRIPTIONS) if p in paths]

    backend_rows: list[tuple[str, str]] = []
    prefix = "cmd/flyteam-server/"
    for path in paths:
        if not path.startswith(prefix) or not path.endswith(".go") or path.endswith("_test.go"):
            continue
        rel = path.removeprefix(prefix)
        basename = Path(path).name
        desc = BACKEND_PATH_DESCRIPTIONS.get(path) or BACKEND_DESCRIPTIONS.get(rel) or BACKEND_DESCRIPTIONS.get(basename) or "Go 后端模块。"
        backend_rows.append((path, desc))
    return top_rows, file_rows, backend_rows


def frontend_rows(paths: list[str]) -> list[tuple[str, str, str]]:
    rows: list[tuple[str, str, str]] = []
    for folder, group_name in FRONTEND_GROUPS.items():
        prefix = folder + "/"
        files = [Path(p).name for p in paths if p.startswith(prefix) and not p.endswith(".gitkeep")]
        if files:
            rows.append((folder + "/", group_name, ", ".join(sorted(files, key=str.lower))))
    return rows


def main() -> None:
    paths = git_tracked_files()
    top_rows, file_rows, backend_rows = collect_rows(paths)
    front_rows = frontend_rows(paths)

    content: list[str] = []
    content.append("# 项目目录明细\n")
    content.append(
        "> 本文件由 `scripts/update_directory_map.py` 自动生成。每次新增、移动、删除目录或关键文件后，请重新运行脚本并提交本文件。\n"
    )
    content.append("## 更新方式\n")
    content.append("```bash\npython scripts/update_directory_map.py\n```\n")
    content.append("Windows PowerShell：\n")
    content.append("```powershell\npython scripts/update_directory_map.py\n```\n")

    content.append("## 顶层目录与职责\n")
    content.append(table(top_rows) + "\n")

    content.append("## 关键文件\n")
    content.append(table(file_rows) + "\n")

    content.append("## 前端资源明细\n")
    content.append("| 目录 | 类型 | 当前文件 |\n| --- | --- | --- |")
    for folder, group_name, files in front_rows:
        content.append(f"| `{folder}` | {group_name} | {files} |")
    content.append("")

    content.append("## Go 后端模块明细\n")
    content.append(table(backend_rows) + "\n")

    content.append("## 运行时/本地文件说明\n")
    content.append(table(RUNTIME_NOTES) + "\n")

    content.append("## 当前 Git 跟踪文件树\n")
    content.append("```text\n" + build_tree(paths) + "\n```\n")

    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    OUTPUT.write_text("\n".join(content), encoding="utf-8")
    print(f"updated {OUTPUT.relative_to(ROOT).as_posix()}")


if __name__ == "__main__":
    main()
