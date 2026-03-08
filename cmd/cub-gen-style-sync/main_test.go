package main

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/confighub/cub-gen/internal/registry"
)

func TestSyncStylesGeneratesAllKindsAcrossThreeStyles(t *testing.T) {
	root := filepath.Join(t.TempDir(), "triple-styles")
	if err := syncStyles(root); err != nil {
		t.Fatalf("syncStyles: %v", err)
	}

	kinds := registry.Kinds()
	if len(kinds) == 0 {
		t.Fatal("expected non-empty generator kind list")
	}

	for _, kind := range kinds {
		name := string(kind)
		t.Run(name, func(t *testing.T) {
			styleAPath := filepath.Join(root, "style-a-yaml", name+".yaml")
			styleBPath := filepath.Join(root, "style-b-markdown", name+".md")
			styleCYaml := filepath.Join(root, "style-c-yaml-plus-docs", name, "triple.yaml")
			styleCMd := filepath.Join(root, "style-c-yaml-plus-docs", name, "triple.md")

			assertFileContains(t, styleAPath, `kind: "`+name+`"`)
			assertFileContains(t, styleBPath, "# "+name+" Triple")
			assertFileContains(t, styleCYaml, `kind: "`+name+`"`)
			assertFileContains(t, styleCMd, "# "+name+" Triple")
		})
	}

	indexPath := filepath.Join(root, "README.md")
	index := mustReadFile(t, indexPath)
	for _, kind := range kinds {
		name := string(kind)
		if !strings.Contains(index, "style-a-yaml/"+name+".yaml") {
			t.Fatalf("index missing style A link for %s", name)
		}
		if !strings.Contains(index, "style-b-markdown/"+name+".md") {
			t.Fatalf("index missing style B link for %s", name)
		}
		if !strings.Contains(index, "style-c-yaml-plus-docs/"+name+"/") {
			t.Fatalf("index missing style C link for %s", name)
		}
	}
}

func TestCheckedInTripleStylesAreInSync(t *testing.T) {
	generated := filepath.Join(t.TempDir(), "triple-styles")
	if err := syncStyles(generated); err != nil {
		t.Fatalf("syncStyles: %v", err)
	}

	checkedIn := filepath.Join(repoRoot(t), "docs", "triple-styles")
	if _, err := os.Stat(checkedIn); err != nil {
		t.Fatalf("expected checked-in triple styles at %s: %v", checkedIn, err)
	}

	generatedFiles := mustListFiles(t, generated)
	checkedInFiles := mustListFiles(t, checkedIn)
	if !equalStringSlices(generatedFiles, checkedInFiles) {
		t.Fatalf("checked-in triple styles file list differs\nwant=%v\ngot=%v", generatedFiles, checkedInFiles)
	}

	for _, rel := range generatedFiles {
		want := mustReadFile(t, filepath.Join(generated, rel))
		got := mustReadFile(t, filepath.Join(checkedIn, rel))
		if got != want {
			t.Fatalf("checked-in triple styles content drift at %s\nrun: go run ./cmd/cub-gen-style-sync", rel)
		}
	}
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(b), want) {
		t.Fatalf("file %s missing %q", path, want)
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func mustListFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	sort.Strings(out)
	return out
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root from %s", file)
		}
		dir = parent
	}
}
