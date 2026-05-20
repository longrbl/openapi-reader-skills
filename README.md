# openapi-reader-skills

A lightweight, zero-dependency OpenAPI/Swagger spec reader and writer — designed for AI-assisted API integration.

---

## Features

- **Zero-dependency single exe** — built with Go, no Python or pip required
- **Subcommand-based CLI** — `list`, `search`, `endpoint`, `schema`, `uses-field`
- **Write support** — `upsert-endpoint`, `remove-endpoint`, `upsert-schema` with built-in quality validation
- **Lazy `$ref` resolution** — fast, no deep copy on large specs
- **Output control** — `--limit`, `--compact`, `--depth` prevent AI context overflow
- **JSON output** — `--output json` for machine parsing
- **Cross-platform** — Windows, Linux, macOS; handles MSYS2 paths, BOM, UTF-8, CJK
- **Safe writes** — automatic `.bak` backup, cross-device rename fallback
- **Quality validation** — `[!]` blocking / `[~]` suggestion warnings on write

## Install

Download the latest `openapi.exe` from releases, or build from source:

```bash
cd go
go build -o openapi.exe .
```

## Usage

```bash
# Query commands
openapi list <spec> [--limit 20] [--compact] [--output json]
openapi search <spec> <keyword> [--limit 10]
openapi tag <spec> <tag>
openapi endpoint <spec> --path /users --method post [--depth 3]
openapi schema <spec> User [--fields-only]
openapi info <spec>
openapi uses-field <spec> <field_name>

# Write commands
openapi upsert-endpoint <spec> --path /users --method POST --file endpoint.json
openapi remove-endpoint <spec> --path /users --method GET
openapi upsert-schema <spec> --name User --file user.json

# Help
openapi help <command>
```

## Examples

```bash
# List all endpoints (limit to 20 to avoid overflow)
openapi list api.json --limit 20

# Get endpoint details with schema depth limit
openapi endpoint api.json --path /users --method post --depth 2

# Search by keyword
openapi search api.json "user" --compact

# Find which endpoints reference a field
openapi uses-field api.json "email"

# Add a new endpoint
openapi upsert-endpoint api.json --path /api/groups --method POST --summary "创建分组" --tag-param "Groups"

# Add a new schema
openapi upsert-schema api.json --name GroupDto --file schema.json
```

## Design

This tool is the Go rewrite of the original Python scripts (`openapi-query.py` / `openapi-writer.py`). It is used as a **skill** by [opencode](https://opencode.ai) for reading and updating OpenAPI specifications without loading the entire file into AI context.

### vs Python version

| Area | Python | Go |
|------|--------|----|
| Runtime | Requires Python 3 + pip | Single binary |
| Dependencies | `orjson` (opt), `pyyaml` (opt) | Zero |
| Output control | None | `--limit`, `--compact`, `--depth`, `--output json`, `--ascii` |
| Validation | Same | Same |
| Write safety | None | `.bak` backup, cross-device fallback |
