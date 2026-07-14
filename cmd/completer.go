package cmd

import (
	"os"
	"path/filepath"
	"strings"
)

// pathCompleter completes filesystem paths for readline.
// Single-path prompt: whole line is the path token, so paths with spaces
// work without quoting.
//
// Interface contract (per readline.AutoCompleter):
//   Do("g", 1) => ["o", "it", "it-shell", "rep"], 1
// Candidates are suffixes beyond the shared prefix; length = shared prefix len.
// readline appends candidates via buf.WriteRunes (no replace).
type pathCompleter struct{}

func (pathCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	path := strings.TrimSpace(string(line[:pos]))
	// strip a leading quote so quoted paths also complete
	if strings.HasPrefix(path, `"`) {
		path = strings.TrimPrefix(path, `"`)
	}

	dir, prefix := filepath.Split(path)
	expanded := expandHome(dir)
	if expanded == "" {
		expanded = "."
	}

	entries, err := os.ReadDir(expanded)
	if err != nil {
		return nil, 0
	}

	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		suffix := name[len(prefix):]
		if e.IsDir() {
			suffix += string(filepath.Separator)
		}
		newLine = append(newLine, []rune(suffix))
	}
	return newLine, len(prefix)
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
