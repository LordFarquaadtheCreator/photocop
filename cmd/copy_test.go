package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildName_Basic(t *testing.T) {
	mt := time.Date(2024, 6, 18, 10, 57, 34, 0, time.Local)
	used := map[string]bool{}
	dir := t.TempDir()

	got := buildName(mt, "photo.JPG", dir, used)
	want := "2024-06-18@10.57.34.JPG"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if !used[want] {
		t.Errorf("want %q marked used", want)
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
		t.Errorf("second = %q, want _2 suffix", second)
	}
	if third != "2024-06-18@10.57.34_3.JPG" {
		t.Errorf("third = %q, want _3 suffix", third)
	}
}

func TestBuildName_CollisionWithExistingFile(t *testing.T) {
	mt := time.Date(2024, 6, 18, 10, 57, 34, 0, time.Local)
	dir := t.TempDir()

	// pre-create the target file so buildName must skip it
	existing := filepath.Join(dir, "2024-06-18@10.57.34.JPG")
	if err := os.WriteFile(existing, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	used := map[string]bool{}
	got := buildName(mt, "new.JPG", dir, used)
	if got != "2024-06-18@10.57.34_2.JPG" {
		t.Errorf("got %q, want _2 (skip existing file)", got)
	}
}

func TestBuildName_PreservesExtensionCase(t *testing.T) {
	mt := time.Date(2024, 8, 25, 4, 45, 6, 0, time.Local)
	dir := t.TempDir()

	cases := []struct{ in, wantExt string }{
		{"clip.MOV", ".MOV"},
		{"raw.nef", ".nef"},
		{"weird.tar.gz", ".gz"},
	}
	for _, c := range cases {
		used := map[string]bool{}
		got := buildName(mt, c.in, dir, used)
		if filepath.Ext(got) != c.wantExt {
			t.Errorf("%s: got ext %q, want %q", c.in, filepath.Ext(got), c.wantExt)
		}
	}
}

func TestCopyFile_PreservesMtime(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	src := filepath.Join(srcDir, "src.txt")
	dst := filepath.Join(dstDir, "dst.txt")

	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	mt := time.Date(2024, 1, 2, 3, 4, 5, 0, time.Local)
	if err := os.Chtimes(src, mt, mt); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(src, dst, mt); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Errorf("content = %q, want hello", string(data))
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(mt) {
		t.Errorf("mtime = %v, want %v", info.ModTime(), mt)
	}
}
