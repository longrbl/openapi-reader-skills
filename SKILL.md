---
name: openapi-reader
description: Use when dealing with any JSON spec file, especially OpenAPI/Swagger specifications. Efficiently queries the spec to return only relevant endpoint details (parameters, request/response schemas) instead of reading the entire file. Also supports writing/updating the spec to keep docs in sync with code. Supports OpenAPI 2.0 (Swagger) and 3.x. Trigger keywords: OpenAPI, Swagger, API对接, API集成, 接口文档, openapi.json, swagger.json, swagger.yaml, API spec, 第三方接口, REST API, api doc, JSON文件, 接口描述文件, 规范文件, 读取接口, 查询接口, 接口定义, 接口文档读写, 规约文件, spec文件, schema, API规范, API定义, JSON读写, json文件操作, 读json, 写json, 修改接口文档, 更新接口文档.
---

# OpenAPI 规范 — 读写一体工具 (Go 版)

你是 API 对接专家。**任何时候**遇到 `openapi.json` / `swagger.json` / 接口规范文件（JSON 格式），**永远不要直接 Read 整个文件**——文件可能数千行且含有大量 `$ref` 引用——始终使用 `openapi.exe` 按需查询。

**开发新 API 或修改接口时**，使用 `openapi.exe` 写入子命令同步更新接口文档，保持代码与文档一致。

> 即使你只是想"看一眼某个接口定义"或"改一下某个字段"，也应该加载本技能，用专用工具比手动 grep+read 更可靠（自动解析 `$ref`、聚合 schema）。

## 一、核心工具

```bash
<skill_dir>/openapi.exe <command> <spec_file> [options]
```

`<skill_dir>` 为该 SKILL.md 所在目录的绝对路径。**单文件 exe，零依赖，直接运行。**

## 二、标准工作流

```
Step 1. 发现接口 — list / search 快速定位目标
Step 2. 获取详情 — endpoint --path /xx --method xx 获取完整请求/响应 Schema
Step 3. 编码实现 — 基于结构化输出直接编写类型定义和调用代码
```

### Step 1 — 发现接口

```bash
# 查看所有接口（方法 + 路径 + 摘要）
openapi.exe list api.json

# 按关键词搜索（匹配路径、摘要、描述、Tag）
openapi.exe search api.json "user"

# 按 Tag 筛选
openapi.exe tag api.json Users

# 分组查看（按 Tag 分组）
openapi.exe list api.json --group-by-tag

# 限制输出条数（防止大文件卡死）
openapi.exe list api.json --limit 20

# 紧凑模式（每接口一行）
openapi.exe list api.json --compact
```

### Step 2 — 获取详情

```bash
# 获取单个接口完整的请求参数 + 响应体（$ref 全部展开）
openapi.exe endpoint api.json --path /users --method post

# 获取某个 Schema/Definition 定义
openapi.exe schema api.json User

# 查看 API 基本信息
openapi.exe info api.json

# 限制 Schema 展开深度（防止深嵌套卡死）
openapi.exe endpoint api.json --path /users --method post --depth 3

# 只输出指定部分
openapi.exe endpoint api.json --path /users --method post --params-only
openapi.exe endpoint api.json --path /users --method post --request-only
openapi.exe endpoint api.json --path /users --method post --response-only
openapi.exe endpoint api.json --path /users --method post --fields-only

# 批量查询多个路径
openapi.exe endpoint api.json --paths /users,/roles --method get
```

### Step 3 — 编码实现

输出中已包含：
- **路径参数 / 查询参数 / Header 参数**（名称、类型、必填、默认值、枚举值、说明）
- **Request Body** 完整 Schema（嵌套对象递归展开，`*` 标记必填字段）
- **各状态码 Response** Schema（200/201/400/401/500… 每个独立列出）
- **接口元信息**（summary、description、tags、operationId、deprecated）
- 自动处理 `$ref`、`allOf`、`oneOf`、`anyOf`、`additionalProperties`
- 自动适配 OpenAPI 2.0（Swagger）和 3.x 两种格式

## 三、所有命令速查

### 查询命令

| 命令 | 语法 | 用途 |
|------|------|------|
| `list` | `openapi.exe list <spec> [options]` | 列出所有接口 |
| `info` | `openapi.exe info <spec> [--output json]` | API 基本信息 |
| `search` | `openapi.exe search <spec> <keyword> [options]` | 按关键词搜索接口 |
| `tag` | `openapi.exe tag <spec> <tag> [options]` | 按 Tag 列出接口 |
| `endpoint` | `openapi.exe endpoint <spec> --path P --method M [options]` | 获取接口详情 |
| `schema` | `openapi.exe schema <spec> <name> [options]` | 获取 Schema 定义 |
| `uses-field` | `openapi.exe uses-field <spec> <field> [options]` | 查找引用某字段的接口 |

### 查询通用参数

| 参数 | 说明 |
|------|------|
| `--limit N` | 限制输出条数（防止大文件卡死） |
| `--compact` | 紧凑模式，每接口一行 |
| `--depth N` | Schema 递归展开最大深度（0=无限） |
| `--output json` | JSON 格式输出（机器可解析） |
| `--group-by-tag` | 按 Tag 分组显示（仅 list） |

### endpoint 专用参数

| 参数 | 说明 |
|------|------|
| `--path P` | API 路径（必填） |
| `--method M` | HTTP 方法（必填） |
| `--paths P1,P2,...` | 批量查询多个路径 |
| `--params-only` | 仅显示参数部分 |
| `--request-only` | 仅显示请求体部分 |
| `--response-only` | 仅显示响应部分 |
| `--fields-only` | 仅显示字段名+类型，不展开嵌套 |

### 写入命令

| 命令 | 语法 | 用途 |
|------|------|------|
| `upsert-endpoint` | `openapi.exe upsert-endpoint <spec> --path P --method M [options]` | 新增或更新接口 |
| `remove-endpoint` | `openapi.exe remove-endpoint <spec> --path P [--method M]` | 删除接口 |
| `upsert-schema` | `openapi.exe upsert-schema <spec> --name N --file F` | 新增或更新 Schema |

### 写入通用参数

| 参数 | 说明 |
|------|------|
| `--diff` | 仅显示 `[!]` 阻断级警告 |

## 四、写入接口文档

开发新 API 时，使用 `openapi.exe` 同步更新 `openapi.json`。

### 新增/更新接口

方式一：通过 JSON 文件（推荐，避免 shell 转义问题）

```powershell
# 1. 写入临时文件
Set-Content -Path "$env:TEMP\new_endpoint.json" -Value @'
{
  "tags": ["AgentGroups"],
  "summary": "创建坐席组",
  "description": "创建一个新的坐席组，包含坐席分配策略、外呼参数和 SIP 出口绑定。",
  "operationId": "createAgentGroup",
  "parameters": [],
  "requestBody": {
    "required": true,
    "content": {
      "application/json": {
        "schema": { "$ref": "#/components/schemas/CreateAgentGroupRequest" }
      }
    }
  },
  "responses": {
    "201": {
      "description": "创建成功",
      "content": {
        "application/json": {
          "schema": { "$ref": "#/components/schemas/AgentGroupDto" }
        }
      }
    }
  }
}
'@ -Encoding default

# 2. 写入 spec
openapi.exe upsert-endpoint openapi.json --path /api/AgentGroups --method POST --file "$env:TEMP\new_endpoint.json"
```

方式二：通过命令行参数（简单接口）

```bash
openapi.exe upsert-endpoint openapi.json \
  --path /api/AgentGroups \
  --method POST \
  --summary "创建坐席组" \
  --description "创建新坐席组" \
  --tag-param "AgentGroups"
```

### 新增/更新 Schema

```powershell
# 通过 JSON 文件
Set-Content -Path "$env:TEMP\schema.json" -Value @'
{
  "type": "object",
  "required": ["id"],
  "properties": {
    "id":   { "type": "string", "format": "uuid", "description": "唯一标识（必填）" },
    "name": { "type": "string", "description": "名称（选填）" }
  }
}
'@ -Encoding default

openapi.exe upsert-schema openapi.json --name MyDto --file "$env:TEMP\schema.json"
```

### 删除接口

```bash
# 删除单个方法
openapi.exe remove-endpoint openapi.json --path /api/AgentGroups --method DELETE

# 删除整个路径的所有方法
openapi.exe remove-endpoint openapi.json --path /api/old-endpoint
```

### 开发中实时写入流程

```
1. 编码 handler → 2. 同步写 openapi.json（同时或紧随其后）
```

## 五、接口文档质量标准

写入接口文档时，必须遵守以下规范。工具会自动校验并输出 `[!]`（阻断）和 `[~]`（建议）警告。

### 1. 每个接口必须有的能力说明

```json
{
  "summary": "创建坐席组",
  "description": "创建一个新的坐席组，包含坐席分配策略、外呼参数和 SIP 出口绑定。支持设置路由模式、并发限制、排队策略等。",
  "operationId": "createAgentGroup"
}
```

### 2. 每个 Schema 必须标记必填字段

```json
{
  "type": "object",
  "required": ["id", "name"],
  "properties": {
    "id":   { "type": "string", "format": "uuid", "description": "坐席组唯一标识（必填）" },
    "name": { "type": "string", "description": "坐席组名称（必填）" }
  }
}
```

### 3. 所有字段必须有中文备注

覆盖范围：路径参数、Query参数、Header参数、Request Body 所有字段、Response 所有字段。

### 4. 枚举/整型/位掩码字段必须在 description 中说明每个值含义

```json
{
  "type": "integer",
  "description": "坐席状态 0=离线 1=空闲 2=忙碌 3=事后处理 4=小休"
}
```

位掩码字段标注「位掩码，可组合」：
```json
{
  "type": "integer",
  "description": "脱敏适用范围（位掩码，可组合）0=None 1=坐席实时 2=管理活跃通话 4=管理通话历史 7=全部"
}
```

### 5. 所有状态码必须有明确的说明

### 6. 校验说明

写入时工具输出如下警告，**所有 `[!]` 标记必须修复后才能提交**：

| 标记 | 校验规则 | 处理要求 |
|------|----------|----------|
| `[!]` | 字段 description 为空 | **必须**补全中文说明 |
| `[!]` | 对象 Schema 有 `properties` 但缺少 `required` 定义 | **必须**添加 `required` 数组 |
| `[!]` | `required` 中某字段在 `properties` 中不存在 | **必须**修正 |
| `[!]` | `enum` 有枚举值但 description 中无 `数值=含义` 映射 | **必须**补充 |
| `[~]` | description 不含中文 | **建议**添加中文备注 |
| `[~]` | 整型字段的 description 中未包含枚举值映射 | **建议**补充 |
| `[~]` | 必填字段的 description 未标注「（必填）」 | **建议**追加 |
| `[~]` | 选填字段的 description 未标注「（选填）」 | **建议**追加 |

## 六、执行规则

1. **永远不要直接 Read OpenAPI 规范文件**，始终使用 `openapi.exe` 查询
2. 先用 `list` / `search` 发现目标接口，再用 `endpoint --path --method` 获取详情
3. 如需对接多个接口，使用多个 bash 调用并行查询（每行一个独立查询）
4. 输出中 `*` 标记的是必填字段，`?` 是 Query 参数，`:` 是 Path 参数
5. 接口详情输出已经包含 Response Schema，无需再单独查询 Schema
6. 大文件使用 `--limit` / `--compact` / `--depth` 防止输出过大导致 opencode 卡死
7. 如需机器解析输出，使用 `--output json`

### 对比 Python 版的新增能力

| 功能 | 说明 |
|------|------|
| 单文件 exe | 零依赖，无需 Python/pip |
| `--limit N` | 限制输出条数，防止大文件撑爆上下文 |
| `--compact` | 紧凑模式，每接口一行 |
| `--depth N` | 限制 Schema 递归展开深度 |
| `--output json` | 机器可解析的 JSON 输出 |
| `--params-only` / `--request-only` / `--response-only` | 按需输出 |
| `--fields-only` | 仅输出字段名和类型 |
| `uses-field` | 查找哪些接口引用了指定字段 |
| `--paths` | 批量查询多个路径 |
| `--group-by-tag` | 按 Tag 分组列出接口 |
| `--diff` | 写入时仅显示阻断级警告 |

### PowerShell 中 `$ref` 的处理

PowerShell 中 `$ref` 会被解释为变量引用，建议使用单引号 here-string `@'...'@` 包裹 JSON 内容：

```powershell
Set-Content -Path schema.json -Value @'
{
  "schema": { "$ref": "#/components/schemas/CreateRequest" }
}
'@
```
