package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Tool inputs ---

type CopyFilesInput struct {
	Src    string `json:"src" jsonschema:"required,Source directory to copy files from"`
	Dst    string `json:"dst" jsonschema:"required,Destination directory to copy files into"`
	DryRun bool   `json:"dry_run" jsonschema:"If true, preview without copying"`
}

// --- Tool outputs ---

type FileMapping struct {
	Original string `json:"original"`
	NewName  string `json:"new_name"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
}

type CopyFilesResult struct {
	Copied  int           `json:"copied"`
	Skipped int           `json:"skipped"`
	Total   int           `json:"total"`
	Files   []FileMapping `json:"files"`
	DryRun  bool          `json:"dry_run"`
}

// Run starts the stdio MCP server.
func Run() error {
	server := mcp.NewServer(&mcp.Implementation{Name: "photocop-mcp", Version: "1.0.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "copy_files",
		Description: "Copy all files from a source directory to a destination directory, renaming each to YYYY-MM-DD@HH.MM.SS.EXT based on its mtime. Collisions get _2, _3, etc. before the extension. Hidden files (dot-prefixed) are skipped. Set dry_run=true to preview without copying.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args CopyFilesInput) (*mcp.CallToolResult, CopyFilesResult, error) {
		return handleCopyFiles(ctx, req, args)
	})

	return server.Run(context.Background(), &mcp.StdioTransport{})
}

// --- Handlers ---

func handleCopyFiles(ctx context.Context, req *mcp.CallToolRequest, args CopyFilesInput) (*mcp.CallToolResult, CopyFilesResult, error) {
	if args.Src == "" {
		return nil, CopyFilesResult{}, fmt.Errorf("src is required")
	}
	if args.Dst == "" {
		return nil, CopyFilesResult{}, fmt.Errorf("dst is required")
	}

	src := expandHome(args.Src)
	dst := expandHome(args.Dst)

	srcInfo, err := os.Stat(src)
	if err != nil {
		return nil, CopyFilesResult{}, fmt.Errorf("source: %w", err)
	}
	if !srcInfo.IsDir() {
		return nil, CopyFilesResult{}, fmt.Errorf("source is not a directory")
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return nil, CopyFilesResult{}, fmt.Errorf("destination: %w", err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return nil, CopyFilesResult{}, fmt.Errorf("read source: %w", err)
	}

	var files []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			files = append(files, e)
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })

	result := CopyFilesResult{Total: len(files), DryRun: args.DryRun}
	used := map[string]bool{}

	for _, e := range files {
		srcPath := filepath.Join(src, e.Name())
		info, err := os.Stat(srcPath)
		if err != nil {
			result.Files = append(result.Files, FileMapping{
				Original: e.Name(),
				Status:   "skipped",
				Error:    err.Error(),
			})
			result.Skipped++
			continue
		}
		name := buildName(info.ModTime(), e.Name(), dst, used)
		mapping := FileMapping{Original: e.Name(), NewName: name}

		if args.DryRun {
			mapping.Status = "preview"
		} else {
			dstPath := filepath.Join(dst, name)
			if err := copyFile(srcPath, dstPath, info.ModTime()); err != nil {
				mapping.Status = "failed"
				mapping.Error = err.Error()
				result.Skipped++
				result.Files = append(result.Files, mapping)
				continue
			}
			mapping.Status = "copied"
		}
		result.Copied++
		result.Files = append(result.Files, mapping)
	}

	return jsonResult(result)
}

// buildName produces YYYY-MM-DD@HH.MM.SS.EXT, appending _N before extension on collision.
// Marks the chosen name in used so repeated calls in one run stay unique.
func buildName(t time.Time, original, dstDir string, used map[string]bool) string {
	base := t.Format("2006-01-02@15.04.05")
	ext := filepath.Ext(original)
	candidate := base + ext
	if !used[candidate] {
		if _, err := os.Stat(filepath.Join(dstDir, candidate)); os.IsNotExist(err) {
			used[candidate] = true
			return candidate
		}
	}
	for n := 2; ; n++ {
		candidate = fmt.Sprintf("%s_%d%s", base, n, ext)
		if !used[candidate] {
			if _, err := os.Stat(filepath.Join(dstDir, candidate)); os.IsNotExist(err) {
				used[candidate] = true
				return candidate
			}
		}
	}
}

func copyFile(src, dst string, mtime time.Time) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	// preserve mtime so future runs stay stable
	return os.Chtimes(dst, mtime, mtime)
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			if p == "~" {
				return home
			}
			if strings.HasPrefix(p, "~/") {
				return filepath.Join(home, p[2:])
			}
		}
	}
	return p
}

// jsonResult marshals the structured output as pretty JSON in the text content.
func jsonResult[T any](out T) (*mcp.CallToolResult, T, error) {
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, out, fmt.Errorf("marshal result: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}, out, nil
}
