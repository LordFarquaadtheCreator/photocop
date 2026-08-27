# photocop

Single binary, two modes:
- `photocop copy` — interactive CLI, copies files dir-to-dir, renaming each to `YYYY-MM-DD@HH.MM.SS.EXT` by mtime.
- `photocop mcp` — stdio MCP server exposing the `copy_files` tool (same rename logic, JSON output).

## Tools (MCP mode)

- `copy_files` — copy files from `src` to `dst`, renaming by mtime. Set `dry_run=true` to preview. Returns JSON with per-file mappings.

## Build

```bash
go build -o photocop .
```

## Run

```bash
./photocop copy --src <dir> --dst <dir> [--dry-run]
./photocop mcp
```

## Key files

| File | Purpose |
|---|---|
| `main.go` | CLI entry point |
| `cmd/root.go` | Cobra root command |
| `cmd/copy.go` | `copy` subcommand + rename/copy logic |
| `cmd/completer.go` | Path autocomplete for interactive prompts |
| `cmd/mcp.go` | `mcp` subcommand — runs the MCP server |
| `internal/mcpserver/server.go` | MCP server, tool handler |
| `internal/mcpserver/server_test.go` | Handler + logic tests |
| `mcp-config.json` | MCP config snippet |
| `build.sh` | macOS .app bundle packaging script |
| `dist_template/` | .app bundle template (Info.plist, launcher) |
