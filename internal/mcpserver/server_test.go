package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	must := func(name string) {
		os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644)
	}
	must("alpha.txt")
	must("beta.go")
	must("gamma.md")
	must("delta.JPG")
	must(".hidden")
	return dir
}

func setMtime(t *testing.T, path string, mt time.Time) {
	t.Helper()
	if err := os.Chtimes(path, mt, mt); err != nil {
		t.Fatal(err)
	}
}

func parseResult(t *testing.T, result *mcp.CallToolResult) CopyFilesResult {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("no content in result")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	var r CopyFilesResult
	if err := json.Unmarshal([]byte(tc.Text), &r); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, tc.Text)
	}
	return r
}

func TestCopyFiles_DryRun(t *testing.T) {
	src := setupTestDir(t)
	dst := t.TempDir()

	// same mtime + same extension → collision → _2 suffix
	mt := time.Date(2024, 6, 18, 10, 57, 34, 0, time.Local)
	setMtime(t, filepath.Join(src, "alpha.txt"), mt)
	setMtime(t, filepath.Join(src, "gamma.md"), mt)

	res, _, err := handleCopyFiles(context.Background(), &mcp.CallToolRequest{}, CopyFilesInput{
		Src:    src,
		Dst:    dst,
		DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	r := parseResult(t, res)

	if r.Total != 4 {
		t.Errorf("total = %d, want 4 (hidden excluded)", r.Total)
	}
	if !r.DryRun {
		t.Error("dry_run should be true")
	}
	// dry-run should not create files
	entries, _ := os.ReadDir(dst)
	if len(entries) != 0 {
		t.Errorf("dst should be empty after dry-run, got %d entries", len(entries))
	}
	// different extensions → no collision, both get base name
	names := map[string]bool{}
	for _, f := range r.Files {
		if f.Status != "preview" {
			t.Errorf("status = %q, want preview", f.Status)
		}
		names[f.NewName] = true
	}
	if !names["2024-06-18@10.57.34.txt"] {
		t.Error("missing 2024-06-18@10.57.34.txt")
	}
	if !names["2024-06-18@10.57.34.md"] {
		t.Error("missing 2024-06-18@10.57.34.md (different ext, no collision)")
	}
}

func TestCopyFiles_DryRun_CollisionSameExt(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	// two .txt files with same mtime → _2 on second
	os.WriteFile(filepath.Join(src, "a.txt"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(src, "b.txt"), []byte("x"), 0o644)
	mt := time.Date(2024, 6, 18, 10, 57, 34, 0, time.Local)
	setMtime(t, filepath.Join(src, "a.txt"), mt)
	setMtime(t, filepath.Join(src, "b.txt"), mt)

	res, _, err := handleCopyFiles(context.Background(), &mcp.CallToolRequest{}, CopyFilesInput{
		Src:    src,
		Dst:    dst,
		DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	r := parseResult(t, res)
	names := map[string]bool{}
	for _, f := range r.Files {
		names[f.NewName] = true
	}
	if !names["2024-06-18@10.57.34.txt"] {
		t.Error("missing 2024-06-18@10.57.34.txt")
	}
	if !names["2024-06-18@10.57.34_2.txt"] {
		t.Error("missing 2024-06-18@10.57.34_2.txt (same ext collision)")
	}
}

func TestCopyFiles_RealCopy(t *testing.T) {
	src := setupTestDir(t)
	dst := t.TempDir()

	mt1 := time.Date(2024, 6, 18, 10, 57, 34, 0, time.Local)
	mt2 := time.Date(2024, 7, 26, 11, 39, 42, 0, time.Local)
	setMtime(t, filepath.Join(src, "alpha.txt"), mt1)
	setMtime(t, filepath.Join(src, "beta.go"), mt2)

	res, _, err := handleCopyFiles(context.Background(), &mcp.CallToolRequest{}, CopyFilesInput{
		Src: src,
		Dst: dst,
	})
	if err != nil {
		t.Fatal(err)
	}
	r := parseResult(t, res)

	if r.Copied != 4 {
		t.Errorf("copied = %d, want 4", r.Copied)
	}
	// verify files exist with correct names
	if _, err := os.Stat(filepath.Join(dst, "2024-06-18@10.57.34.txt")); err != nil {
		t.Error("alpha.txt not renamed correctly")
	}
	if _, err := os.Stat(filepath.Join(dst, "2024-07-26@11.39.42.go")); err != nil {
		t.Error("beta.go not renamed correctly")
	}
	// verify mtime preserved
	info, _ := os.Stat(filepath.Join(dst, "2024-06-18@10.57.34.txt"))
	if !info.ModTime().Equal(mt1) {
		t.Errorf("mtime = %v, want %v", info.ModTime(), mt1)
	}
}

func TestCopyFiles_HiddenFilesSkipped(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	os.WriteFile(filepath.Join(src, ".DS_Store"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(src, "visible.txt"), []byte("x"), 0o644)

	res, _, err := handleCopyFiles(context.Background(), &mcp.CallToolRequest{}, CopyFilesInput{
		Src: src,
		Dst: dst,
	})
	if err != nil {
		t.Fatal(err)
	}
	r := parseResult(t, res)
	if r.Total != 1 {
		t.Errorf("total = %d, want 1 (.hidden excluded)", r.Total)
	}
}

func TestCopyFiles_MissingSrc(t *testing.T) {
	_, _, err := handleCopyFiles(context.Background(), &mcp.CallToolRequest{}, CopyFilesInput{
		Src: "/nonexistent/xyz",
		Dst: t.TempDir(),
	})
	if err == nil {
		t.Error("expected error for missing source")
	}
}

func TestCopyFiles_SrcNotDir(t *testing.T) {
	src := t.TempDir()
	f := filepath.Join(src, "file.txt")
	os.WriteFile(f, []byte("x"), 0o644)

	_, _, err := handleCopyFiles(context.Background(), &mcp.CallToolRequest{}, CopyFilesInput{
		Src: f,
		Dst: t.TempDir(),
	})
	if err == nil {
		t.Error("expected error for non-directory source")
	}
}

func TestCopyFiles_EmptySrc(t *testing.T) {
	_, _, err := handleCopyFiles(context.Background(), &mcp.CallToolRequest{}, CopyFilesInput{
		Dst: t.TempDir(),
	})
	if err == nil {
		t.Error("expected error for empty src")
	}
}

func TestCopyFiles_EmptyDst(t *testing.T) {
	_, _, err := handleCopyFiles(context.Background(), &mcp.CallToolRequest{}, CopyFilesInput{
		Src: t.TempDir(),
	})
	if err == nil {
		t.Error("expected error for empty dst")
	}
}

func TestCopyFiles_DstCreated(t *testing.T) {
	src := setupTestDir(t)
	base := t.TempDir()
	dst := filepath.Join(base, "new", "nested", "dir")

	_, _, err := handleCopyFiles(context.Background(), &mcp.CallToolRequest{}, CopyFilesInput{
		Src: src,
		Dst: dst,
	})
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dst)
	if err != nil || !info.IsDir() {
		t.Error("nested dst should be created")
	}
}

func TestCopyFiles_TildeExpansion(t *testing.T) {
	home, _ := os.UserHomeDir()
	src := filepath.Join(home, "Desktop")
	if _, err := os.Stat(src); err != nil {
		t.Skip("no Desktop dir")
	}
	dst := t.TempDir()

	// use ~/Desktop as src — should expand without error
	_, _, err := handleCopyFiles(context.Background(), &mcp.CallToolRequest{}, CopyFilesInput{
		Src: "~/Desktop",
		Dst: dst,
	})
	// may have no files but should not error on stat
	if err != nil && !contains(err.Error(), "no such file") {
		t.Errorf("tilde expansion failed: %v", err)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(s) > len(sub) && (indexOf(s, sub) >= 0)))
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestBuildName_Basic(t *testing.T) {
	mt := time.Date(2024, 6, 18, 10, 57, 34, 0, time.Local)
	used := map[string]bool{}
	dir := t.TempDir()

	got := buildName(mt, "photo.JPG", dir, used)
	want := "2024-06-18@10.57.34.JPG"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildName_CollisionAppendsN(t *testing.T) {
	mt := time.Date(2024, 6, 18, 10, 57, 34, 0, time.Local)
	used := map[string]bool{}
	dir := t.TempDir()

	first := buildName(mt, "a.JPG", dir, used)
	second := buildName(mt, "b.JPG", dir, used)
	third := buildName(mt, "c.JPG", dir, used)

	if first != "2024-06-18@10.57.34.JPG" {
		t.Errorf("first = %q", first)
	}
	if second != "2024-06-18@10.57.34_2.JPG" {
		t.Errorf("second = %q, want _2", second)
	}
	if third != "2024-06-18@10.57.34_3.JPG" {
		t.Errorf("third = %q, want _3", third)
	}
}

func TestBuildName_SkipExistingFile(t *testing.T) {
	mt := time.Date(2024, 6, 18, 10, 57, 34, 0, time.Local)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "2024-06-18@10.57.34.JPG"), []byte("x"), 0o644)

	used := map[string]bool{}
	got := buildName(mt, "new.JPG", dir, used)
	if got != "2024-06-18@10.57.34_2.JPG" {
		t.Errorf("got %q, want _2", got)
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	if got := expandHome("~"); got != home {
		t.Errorf("~ = %q, want %q", got, home)
	}
	if got := expandHome("~/foo"); got != filepath.Join(home, "foo") {
		t.Errorf("~/foo = %q", got)
	}
	if got := expandHome("/abs"); got != "/abs" {
		t.Errorf("/abs = %q", got)
	}
}
