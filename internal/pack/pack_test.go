package pack

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestPack_BasicDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "package main")
	writeFile(t, dir, "go.mod", "module example")

	res, err := Pack(dir, "")
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(res.Path) })

	if res.Hash == "" {
		t.Error("expected non-empty hash")
	}
	if res.Size == 0 {
		t.Error("expected non-zero size")
	}

	files := listTarFiles(t, res.Path)
	if !contains(files, "main.go") {
		t.Errorf("expected main.go in archive; got %v", files)
	}
	if !contains(files, "go.mod") {
		t.Errorf("expected go.mod in archive; got %v", files)
	}
}

func TestPack_GitDirAlwaysExcluded(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "package main")
	gitDir := filepath.Join(dir, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, filepath.Join(".git", "HEAD"), "ref: refs/heads/main")

	res, err := Pack(dir, "")
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(res.Path) })

	files := listTarFiles(t, res.Path)
	for _, f := range files {
		if strings.HasPrefix(f, ".git") {
			t.Errorf("expected .git to be excluded, found %s in archive", f)
		}
	}
}

func TestPack_StraitIgnoreRespected(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "package main")
	writeFile(t, dir, "secret.txt", "should be ignored")
	writeFile(t, dir, "node_modules/pkg/index.js", "should be ignored")
	writeFile(t, dir, ".straitignore", "secret.txt\nnode_modules/")

	res, err := Pack(dir, "")
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(res.Path) })

	files := listTarFiles(t, res.Path)
	if contains(files, "secret.txt") {
		t.Error("expected secret.txt to be excluded")
	}
	for _, f := range files {
		if strings.HasPrefix(f, "node_modules") {
			t.Errorf("expected node_modules to be excluded, got %s", f)
		}
	}
	if !contains(files, "main.go") {
		t.Errorf("expected main.go in archive; got %v", files)
	}
}

func TestPack_IgnoreFileOverride(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "package main")
	writeFile(t, dir, "skip.txt", "skip me")

	ignoreFile := filepath.Join(t.TempDir(), ".myignore")
	if err := os.WriteFile(ignoreFile, []byte("skip.txt\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := Pack(dir, ignoreFile)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(res.Path) })

	files := listTarFiles(t, res.Path)
	if contains(files, "skip.txt") {
		t.Error("expected skip.txt to be excluded via explicit ignore file")
	}
}

func TestPack_HashIsDeterministic(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "package a")
	writeFile(t, dir, "b.go", "package b")

	r1, err := Pack(dir, "")
	if err != nil {
		t.Fatalf("Pack 1: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(r1.Path) })

	r2, err := Pack(dir, "")
	if err != nil {
		t.Fatalf("Pack 2: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(r2.Path) })

	if r1.Hash != r2.Hash {
		t.Errorf("expected deterministic hash: %s != %s", r1.Hash, r2.Hash)
	}
}

func TestPack_NonDirectoryError(t *testing.T) {
	t.Parallel()
	f, err := os.CreateTemp(t.TempDir(), "file")
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	_, err = Pack(f.Name(), "")
	if err == nil {
		t.Error("expected error for non-directory source")
	}
}

func TestShouldIgnore_WildcardPattern(t *testing.T) {
	t.Parallel()
	patterns := []ignorePattern{parsePattern("*.log")}
	if !shouldIgnore("app.log", false, patterns) {
		t.Error("expected app.log to be ignored by *.log")
	}
	if shouldIgnore("main.go", false, patterns) {
		t.Error("expected main.go NOT to be ignored by *.log")
	}
}

func TestShouldIgnore_DoubleStarPattern(t *testing.T) {
	t.Parallel()
	patterns := []ignorePattern{parsePattern("build/**")}
	if !shouldIgnore("build/output/bin", false, patterns) {
		t.Error("expected build/output/bin to be ignored by build/**")
	}
	if shouldIgnore("src/main.go", false, patterns) {
		t.Error("expected src/main.go NOT to be ignored by build/**")
	}
}

func TestShouldIgnore_NegationPattern(t *testing.T) {
	t.Parallel()
	patterns := []ignorePattern{
		parsePattern("*.log"),
		parsePattern("!keep.log"),
	}
	if !shouldIgnore("app.log", false, patterns) {
		t.Error("expected app.log to be ignored")
	}
	if shouldIgnore("keep.log", false, patterns) {
		t.Error("expected keep.log NOT to be ignored due to negation")
	}
}

// helpers.

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func listTarFiles(t *testing.T, archivePath string) []string {
	t.Helper()
	f, err := os.Open(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, hdr.Name)
	}
	return names
}

func contains(slice []string, s string) bool {
	return slices.Contains(slice, s)
}
