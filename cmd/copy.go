package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
)

var (
	copySrc string
	copyDst string
	dryRun  bool
)

var copyCmd = &cobra.Command{
	Use:   "copy",
	Short: "Copy files from src to dst, renaming each by mtime.",
	RunE:  runCopy,
}

func init() {
	copyCmd.Flags().StringVarP(&copySrc, "src", "s", "", "source directory")
	copyCmd.Flags().StringVarP(&copyDst, "dst", "d", "", "destination directory")
	copyCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "print plan without copying")
	rootCmd.AddCommand(copyCmd)
}

func runCopy(cmd *cobra.Command, args []string) error {
	if copySrc == "" {
		copySrc = prompt("Source directory: ")
	}
	if copyDst == "" {
		copyDst = prompt("Destination directory: ")
	}
	if copySrc == "" || copyDst == "" {
		return errors.New("source and destination required")
	}

	copySrc = expandHome(copySrc)
	copyDst = expandHome(copyDst)

	srcInfo, err := os.Stat(copySrc)
	if err != nil {
		return fmt.Errorf("source: %w", err)
	}
	if !srcInfo.IsDir() {
		return errors.New("source is not a directory")
	}
	if err := os.MkdirAll(copyDst, 0o755); err != nil {
		return fmt.Errorf("destination: %w", err)
	}

	entries, err := os.ReadDir(copySrc)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}

	var files []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			files = append(files, e)
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })

	if len(files) == 0 {
		fmt.Println("No files in source.")
		return nil
	}

	used := map[string]bool{}
	copied, skipped := 0, 0
	for _, e := range files {
		srcPath := filepath.Join(copySrc, e.Name())
		info, err := os.Stat(srcPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", e.Name(), err)
			skipped++
			continue
		}
		name := buildName(info.ModTime(), e.Name(), copyDst, used)
		dstPath := filepath.Join(copyDst, name)

		if dryRun {
			fmt.Printf("[dry-run] %s -> %s\n", e.Name(), name)
			copied++
			continue
		}
		if err := copyFile(srcPath, dstPath, info.ModTime()); err != nil {
			fmt.Fprintf(os.Stderr, "fail %s: %v\n", e.Name(), err)
			skipped++
			continue
		}
		fmt.Printf("%s -> %s\n", e.Name(), name)
		copied++
	}
	fmt.Printf("\nDone. Copied %d, skipped %d.\n", copied, skipped)
	return nil
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

func prompt(msg string) string {
	// non-interactive stdin: plain scan, no readline
	if fi, err := os.Stdin.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) == 0 {
		fmt.Print(msg)
		var s string
		fmt.Fscanln(os.Stdin, &s)
		return strings.TrimSpace(s)
	}
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          msg,
		HistoryFile:     "",
		AutoComplete:    pathCompleter{},
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Print(msg)
		var s string
		fmt.Fscanln(os.Stdin, &s)
		return strings.TrimSpace(s)
	}
	defer rl.Close()
	line, err := rl.Readline()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(line)
}
