package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// setupCompleterDir makes a temp dir with known entries:
//   alpha/  beta/  alpha.txt  beta.go  gamma.md
func setupCompleterDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	must := func(name string, isDir bool) {
		p := filepath.Join(dir, name)
		if isDir {
			if err := os.Mkdir(p, 0o755); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	must("alpha", true)
	must("beta", true)
	must("alpha.txt", false)
	must("beta.go", false)
	must("gamma.md", false)
	return dir
}

func toStrings(runes [][]rune) []string {
	out := make([]string, len(runes))
	for i, r := range runes {
		out[i] = string(r)
	}
	sort.Strings(out)
	return out
}

func TestCompleter_NoPrefix_ListsAll(t *testing.T) {
	dir := setupCompleterDir(t)
	c := pathCompleter{}
	line := []rune(dir + "/")
	got, length := c.Do(line, len(line))

	want := []string{"alpha.txt", "alpha/", "beta.go", "beta/", "gamma.md"} // "." (0x2E) < "/" (0x2F)
	if !reflect.DeepEqual(toStrings(got), want) {
		t.Errorf("got %v, want %v", toStrings(got), want)
	}
	if length != 0 {
		t.Errorf("length = %d, want 0", length)
	}
}

func TestCompleter_Prefix_FiltersAndReturnsSuffix(t *testing.T) {
	dir := setupCompleterDir(t)
	c := pathCompleter{}
	// line ends with "al" → prefix "al", matches "alpha/" and "alpha.txt"
	// suffixes beyond "al": "pha/", "pha.txt"
	line := []rune(dir + "/al")
	got, length := c.Do(line, len(line))

	want := []string{"pha.txt", "pha/"}
	if !reflect.DeepEqual(toStrings(got), want) {
		t.Errorf("got %v, want %v", toStrings(got), want)
	}
	if length != 2 {
		t.Errorf("length = %d, want 2", length)
	}
}

func TestCompleter_SingleMatch_ReturnsSuffix(t *testing.T) {
	dir := setupCompleterDir(t)
	c := pathCompleter{}
	line := []rune(dir + "/gamma.m")
	got, length := c.Do(line, len(line))

	want := []string{"d"} // suffix beyond "gamma.m"
	if !reflect.DeepEqual(toStrings(got), want) {
		t.Errorf("got %v, want %v", toStrings(got), want)
	}
	if length != 7 {
		t.Errorf("length = %d, want 7", length)
	}
}

func TestCompleter_DirGetsTrailingSeparator(t *testing.T) {
	dir := t.TempDir()
	// uniquely-named dir so prefix matches only it
	if err := os.Mkdir(filepath.Join(dir, "zed"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := pathCompleter{}
	line := []rune(dir + "/z")
	got, length := c.Do(line, len(line))

	want := []string{"ed/"}
	if !reflect.DeepEqual(toStrings(got), want) {
		t.Errorf("got %v, want %v", toStrings(got), want)
	}
	if length != 1 {
		t.Errorf("length = %d, want 1", length)
	}
}

func TestCompleter_NoMatch_Empty(t *testing.T) {
	dir := setupCompleterDir(t)
	c := pathCompleter{}
	line := []rune(dir + "/zzz")
	got, length := c.Do(line, len(line))
	if len(got) != 0 {
		t.Errorf("got %v, want empty", toStrings(got))
	}
	if length != 3 {
		t.Errorf("length = %d, want 3", length)
	}
}

func TestCompleter_BadDir_Empty(t *testing.T) {
	c := pathCompleter{}
	line := []rune("/nonexistent/path/xyz")
	got, _ := c.Do(line, len(line))
	if len(got) != 0 {
		t.Errorf("got %v, want empty for bad dir", toStrings(got))
	}
}

func TestCompleter_EmptyLine_ListsCwd(t *testing.T) {
	c := pathCompleter{}
	got, length := c.Do([]rune(""), 0)
	if length != 0 {
		t.Errorf("length = %d, want 0", length)
	}
	// cwd has entries (test running from module dir); just assert non-empty
	if len(got) == 0 {
		t.Error("expected entries in cwd")
	}
}

func TestCompleter_LeadingQuoteStripped(t *testing.T) {
	dir := setupCompleterDir(t)
	c := pathCompleter{}
	line := []rune(`"` + dir + "/be")
	got, length := c.Do(line, len(line))

	want := []string{"ta.go", "ta/"} // "." (0x2E) sorts before "/" (0x2F)
	if !reflect.DeepEqual(toStrings(got), want) {
		t.Errorf("got %v, want %v", toStrings(got), want)
	}
	if length != 2 {
		t.Errorf("length = %d, want 2", length)
	}
}

func TestCompleter_PathWithSpaces(t *testing.T) {
	base := t.TempDir()
	spaced := filepath.Join(base, "to edit")
	if err := os.Mkdir(spaced, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(spaced, "edited"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(spaced, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := pathCompleter{}
	line := []rune(spaced + "/")
	got, length := c.Do(line, len(line))

	want := []string{"edited/", "notes.txt"}
	if !reflect.DeepEqual(toStrings(got), want) {
		t.Errorf("got %v, want %v", toStrings(got), want)
	}
	if length != 0 {
		t.Errorf("length = %d, want 0", length)
	}
}

func TestCompleter_PathWithSpaces_Prefix(t *testing.T) {
	base := t.TempDir()
	spaced := filepath.Join(base, "to edit")
	os.Mkdir(spaced, 0o755)
	os.Mkdir(filepath.Join(spaced, "edited"), 0o755)
	os.WriteFile(filepath.Join(spaced, "evan.txt"), []byte("x"), 0o644)

	c := pathCompleter{}
	line := []rune(spaced + "/e")
	got, length := c.Do(line, len(line))

	want := []string{"dited/", "van.txt"}
	if !reflect.DeepEqual(toStrings(got), want) {
		t.Errorf("got %v, want %v", toStrings(got), want)
	}
	if length != 1 {
		t.Errorf("length = %d, want 1", length)
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	if got := expandHome("~"); got != home {
		t.Errorf("~ = %q, want %q", got, home)
	}
	if got := expandHome("~/foo"); got != filepath.Join(home, "foo") {
		t.Errorf("~/foo = %q, want %q", got, filepath.Join(home, "foo"))
	}
	if got := expandHome("~user"); got != "~user" {
		t.Errorf("~user = %q, want %q (unsupported, leave as-is)", got, "~user")
	}
	if got := expandHome("/abs/path"); got != "/abs/path" {
		t.Errorf("/abs/path = %q, want /abs/path", got)
	}
	if got := expandHome("rel/path"); got != "rel/path" {
		t.Errorf("rel/path = %q, want rel/path", got)
	}
}
