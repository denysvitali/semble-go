package index

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFiles creates rel:body files under dir.
func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func relSet(files []fileInfo) map[string]bool {
	m := make(map[string]bool, len(files))
	for _, f := range files {
		m[f.rel] = true
	}
	return m
}

func TestSembleignoreExcludesPatterns(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"keep.go":       "package a\n",
		"secret.go":     "package a\n",
		"gen/auto.go":   "package gen\n",
		"sub/secret.go": "package a\n",
		".sembleignore": "# extra ignores\nsecret.go\ngen\n\n",
	})
	il := loadIgnore(dir)
	files, err := walk(dir, map[string]bool{"code": true}, il)
	if err != nil {
		t.Fatal(err)
	}
	got := relSet(files)
	if !got["keep.go"] {
		t.Error("keep.go should be indexed")
	}
	for _, ex := range []string{"secret.go", "sub/secret.go", "gen/auto.go"} {
		if got[ex] {
			t.Errorf("%s should be excluded by .sembleignore", ex)
		}
	}
}

func TestSembleignoreDefaultsStillApply(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"keep.go":           "package a\n",
		"vendor/dep.go":     "package v\n",
		"node_modules/x.js": "var x = 1\n",
	})
	il := loadIgnore(dir) // no .sembleignore present
	files, err := walk(dir, map[string]bool{"code": true}, il)
	if err != nil {
		t.Fatal(err)
	}
	got := relSet(files)
	if !got["keep.go"] {
		t.Error("keep.go should be indexed")
	}
	if got["vendor/dep.go"] || got["node_modules/x.js"] {
		t.Error("default ignores (vendor, node_modules) must still apply")
	}
}
