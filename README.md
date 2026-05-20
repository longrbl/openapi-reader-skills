# openapi-reader-skills

轻量级、零依赖的 OpenAPI/Swagger 规范读写工具 — 专为 AI 辅助 API 集成设计。

---

## 特点

- **单文件 exe，零依赖** — Go 编译，无需 Python 或 pip
- **子命令式 CLI** — `list`、`search`、`endpoint`、`schema`、`uses-field`
- **写入支持** — `upsert-endpoint`、`remove-endpoint`、`upsert-schema`，内置质量校验
- **懒加载 `$ref` 解析** — 快，大文件不深拷贝
- **输出控制** — `--limit`、`--compact`、`--depth` 防止 AI 上下文溢出
- **JSON 输出** — `--output json` 供机器解析
- **跨平台** — Windows、Linux、macOS；处理 MSYS2 路径、BOM、UTF-8、中文编码
- **远程 URL 支持** — 直接传入 `https://` 链接自动抓取，无需手动下载
- **安全写入** — 自动 `.bak` 备份、跨设备 Rename 回退
- **质量校验** — `[!]` 阻断级 / `[~]` 建议级警告

## 安装

从 Releases 下载最新的 `openapi.exe`，或从源码构建：

```bash
cd go
go build -o openapi.exe .
```

## 用法

```bash
# 查询命令
openapi list <spec> [--limit 20] [--compact] [--output json]
openapi search <spec> <keyword> [--limit 10]
openapi tag <spec> <tag>
openapi endpoint <spec> --path /users --method post [--depth 3]
openapi schema <spec> User [--fields-only]
openapi info <spec>
openapi uses-field <spec> <field_name>

# 写入命令
openapi upsert-endpoint <spec> --path /users --method POST --file endpoint.json
openapi remove-endpoint <spec> --path /users --method GET
openapi upsert-schema <spec> --name User --file user.json

# 帮助
openapi help <command>
```

## 示例

```bash
# 列出所有接口（限制 20 条防溢出）
openapi list api.json --limit 20

# 从远程 URL 读取规格
openapi list https://petstore.swagger.io/v2/swagger.json --limit 5

# 获取接口详情，限制 Schema 展开深度
openapi endpoint api.json --path /users --method post --depth 2

# 按关键词搜索
openapi search api.json "user" --compact

# 查找哪些接口引用了某字段
openapi uses-field api.json "email"

# 新增接口
openapi upsert-endpoint api.json --path /api/groups --method POST --summary "创建分组" --tag-param "Groups"

# 新增 Schema
openapi upsert-schema api.json --name GroupDto --file schema.json
```

## 设计说明

本工具是原 Python 脚本（`openapi-query.py` / `openapi-writer.py`）的 Go 重写版。作为 [opencode](https://opencode.ai) 的 **skill** 使用，用于读取和更新 OpenAPI 规范文件，避免将整个文件加载到 AI 上下文中。

### 与 Python 版对比

| 维度 | Python | Go |
|------|--------|----|
| 运行环境 | 需要 Python 3 + pip | 单文件 exe |
| 外部依赖 | `orjson`（可选）、`pyyaml`（可选） | 零 |
| 输出控制 | 无 | `--limit`、`--compact`、`--depth`、`--output json`、`--ascii` |
| 质量校验 | 同上 | 同上 |
| 写入安全 | 无 | `.bak` 备份、跨设备回退 |
