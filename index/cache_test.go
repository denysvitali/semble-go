package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCacheDir(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	_ = os.Unsetenv("SEMBLE_CACHE_DIR")
	dir := CacheDir()
	if dir == "" {
		t.Fatal("CacheDir() is empty")
	}
	if !strings.HasSuffix(dir, "semble-go") {
		t.Fatalf("CacheDir() %q does not end with semble-go", dir)
	}
}

func TestCleanCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	_ = os.Unsetenv("SEMBLE_CACHE_DIR")
	dir := CacheDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dummy.gob"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	removed, freed, err := CleanCache()
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
	if freed != 5 {
		t.Fatalf("freed = %d, want 5", freed)
	}
	if _, err := os.Stat(filepath.Join(dir, "dummy.gob")); !os.IsNotExist(err) {
		t.Fatal("dummy.gob still exists after clean")
	}
}
