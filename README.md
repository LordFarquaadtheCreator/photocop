# photocop

Copy files from one directory to another, renaming each to `YYYY-MM-DD@HH.MM.SS.EXT` based on its mtime. Single Go binary with two modes: an interactive CLI and a stdio MCP server.

## Modes

- `photocop copy` — interactive/flag-driven CLI. Prompts for src/dst if missing, prints `old -> new` lines.
- `photocop mcp` — stdio MCP server exposing the `copy_files` tool with JSON output.

Both share the same rename + collision logic.

## Rename behavior

- Renames each file to `YYYY-MM-DD@HH.MM.SS.EXT` using file mtime, local 24h
- Collisions (same name incl extension) get `_2`, `_3`, ... before extension
- Hidden files (dot-prefixed) skipped
- Files sorted by name for deterministic `_N` assignment
- mtime preserved on copied files (idempotent re-runs)
- Subdirectories inside `src` are skipped — only files copied
- `~` expanded in paths

## CLI

```bash
./photocop copy --src ~/Downloads --dst ~/Photos --dry-run
./photocop copy -s ~/Downloads -d ~/Photos
```

| Flag | Description |
|---|---|
| `--src` / `-s` | Source directory |
| `--dst` / `-d` | Destination directory (created if missing) |
| `--dry-run` / `-n` | Print plan without copying |

If src/dst flags are omitted, prompts interactively with path autocomplete.

## MCP

### `copy_files`

| Param | Type | Required | Description |
|---|---|---|---|
| `src` | string | yes | Source directory |
| `dst` | string | yes | Destination directory (created if missing) |
| `dry_run` | bool | no | Preview without copying |

Returns JSON with `copied`, `skipped`, `total`, `dry_run`, and `files[]` (each with `original`, `new_name`, `status`, optional `error`).

## Build

```bash
go build -o photocop .
```

## macOS .app bundle

```bash
./build.sh   # produces dist/PhotoCopy.app
```

## Dependencies

- Go 1.25+
- `github.com/spf13/cobra` (CLI)
- `github.com/modelcontextprotocol/go-sdk/mcp` (MCP server)
