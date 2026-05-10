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
    "cmd/flyteam-server/": "Go 后端服务，包含路由、鉴权、内容管理、博客社区、RAG、上传等模块。",
    "docs/": "项目文档总目录。",
    "docs/knowledge/": "本地知识库/PDF 草稿占位目录，PDF 默认不提交 Git。",
    "docs/planning/": "规划、路线图、多人协作任务分配。",
    "docs/reports/": "测试、安全、验收报告。",
    "scripts/": "项目维护脚本。",
    "storage/": "运行数据目录：数据库、上传文件、RAG 索引和日志，默认不提交。",
}

FILE_DESCRIPTIONS = {
    ".env.example": "环境变量模板。",
    ".gitignore": "Git 忽略规则，排除密钥、运行数据、上传文件、日志和本地缓存。",
    "CONTRIBUTING.md": "协作流程说明。",
    "README.md": "项目总说明和本地运行指南。",
    "go.mod": "Go 模块定义。",
    "go.sum": "Go 依赖校验锁定文件。",
    "docs/PROJECT_STRUCTURE.md": "目录结构约定和新增文件放置规范。",
    "docs/DIRECTORY_MAP.md": "自动生成的项目目录明细。",
    "docs/planning/blog-community-roadmap.md": "博客社区化改造路线图。",
    "docs/planning/team-task-allocation.md": "z3 / grand / dl 任务分配。",
    "docs/reports/final-test-security-report.md": "功能与安全测试报告。",
    "scripts/update_directory_map.py": "自动刷新 docs/DIRECTORY_MAP.md。",
}

BACKEND_DESCRIPTIONS = {
    "admin_blog_ops.go": "博客站管理员/超级管理员操作、用户审核、审计接口。",
    "auth.go": "宣传站管理员鉴权、会话、角色权限。",
    "blog_site_state.go": "博客站开放/关闭状态与前端访问控制。",
    "cache.go": "缓存控制与辅助逻辑。",
    "captcha.go": "招新报名动态 C 语言验证码。",
    "community_auth.go": "社区鉴权公共逻辑。",
    "community_blog.go": "博客文章发布、编辑、读取、浏览量等。",
    "community_dl_comments.go": "博客评论、点赞、收藏等互动。",
    "community_dl_groups.go": "群聊、群成员、群管理。",
    "community_dl_notify_search.go": "通知与搜索。",
    "community_dl_routes.go": "社区 API 路由分发。",
    "community_dl_social_messages.go": "关注、好友、私信等社交消息。",
    "community_friends.go": "好友申请与好友关系。",
    "community_grand_auth.go": "社区用户注册、登录、资料与账号管理。",
    "community_reserved.go": "社区预留/状态接口。",
    "content.go": "官网内容聚合、排序、奖项/前辈墙/新闻等核心内容逻辑。",
    "content_review_recruit.go": "团队回顾、相册、招新报名数据处理。",
    "database.go": "SQLite 初始化、表结构迁移、默认账号/数据迁移。",
    "main.go": "服务入口、配置加载、HTTP 路由、静态文件服务和安全响应头。",
    "rag.go": "RAG 知识库、PDF 文本提取、向量检索、问答调用。",
    "upload.go": "PDF、图片、头像等上传处理和文件安全校验。",
}

FRONTEND_GROUPS = {
    "app/static/pages": "页面模板",
    "app/static/js": "前端脚本",
    "app/static/css": "样式文件",
}

RUNTIME_NOTES = [
    (".env", "本地/服务器真实环境变量，包含密钥，禁止提交。"),
    ("storage/flyteam.db", "SQLite 运行数据库，保存账号、内容、文章、聊天等数据。"),
    ("storage/uploads/", "后台上传图片、头像、PDF、博客图片等缓存。"),
    ("storage/chroma/", "旧版 Chroma 向量库缓存，如存在则不提交。"),
    ("storage/*.json", "兼容旧版 JSON 数据和迁移来源，不提交。"),
    ("storage/*.log", "运行日志，不提交。"),
    (".venv/", "本地 Python 虚拟环境，不提交。"),
    ("archive/legacy-python/*.codex_backup", "旧 Python 备份文件，不提交。"),
]


def git_tracked_files() -> list[str]:
    # Include tracked files plus new untracked files that are not ignored, so this
    # document can be generated before the first commit that introduces them.
    result = subprocess.run(
        ["git", "ls-files", "--cached", "--others", "--exclude-standard"],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
        encoding="utf-8",
    )
    return sorted(line.strip().replace("\\", "/") for line in result.stdout.splitlines() if line.strip())


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
    for path in paths:
        prefix = "cmd/flyteam-server/"
        if not path.startswith(prefix) or not path.endswith(".go") or path.endswith("_test.go"):
            continue
        name = path.removeprefix(prefix)
        backend_rows.append((path, BACKEND_DESCRIPTIONS.get(name, "Go 后端模块。")))
    return top_rows, file_rows, backend_rows


def frontend_rows(paths: list[str]) -> list[tuple[str, str, str]]:
    rows: list[tuple[str, str, str]] = []
    for folder, group_name in FRONTEND_GROUPS.items():
        prefix = folder + "/"
        files = [Path(p).name for p in paths if p.startswith(prefix) and not p.endswith(".gitkeep")]
        if files:
            rows.append((folder + "/", group_name, ", ".join(files)))
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
