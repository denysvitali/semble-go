package index

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIsRemote(t *testing.T) {
	cases := []struct {
		spec string
		want bool
	}{
		{"https://github.com/foo/bar", true},
		{"http://example.com/repo", true},
		{"git://example.com/repo", true},
		{"ssh://git@github.com/foo/bar", true},
		{"git@github.com:foo/bar.git", true},
		{".", false},
		{"./local", false},
		{"/abs/path", false},
		{"relative/path", false},
	}
	for _, c := range cases {
		if got := IsRemote(c.spec); got != c.want {
			t.Errorf("IsRemote(%q) = %v, want %v", c.spec, got, c.want)
		}
	}
}

func TestResolveRepoLocal(t *testing.T) {
	dir := t.TempDir()
	got, err := ResolveRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Errorf("ResolveRepo(%q) = %q, want same", dir, got)
	}
}

func TestResolveRepoLocalGit(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "init", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	// Create a file and commit so HEAD exists.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "-C", dir, "add", ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	cmd = exec.Command("git", "-C", dir, "commit", "-m", "init")
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}

	got, err := ResolveRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Errorf("ResolveRepo(%q) = %q, want same", dir, got)
	}
}

func TestResolveRepoFileURL(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "init", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "-C", dir, "add", ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	cmd = exec.Command("git", "-C", dir, "commit", "-m", "init")
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}

	fileURL := "file://" + dir
	got, err := ResolveRepo(fileURL)
	if err != nil {
		t.Fatalf("ResolveRepo(%q): %v", fileURL, err)
	}
	// Should be a path in the cache dir.
	if got == fileURL {
		t.Error("expected resolved path, got back the URL")
	}
	// Verify the cloned repo has the file.
	if _, err := os.Stat(filepath.Join(got, "README.md")); err != nil {
		t.Errorf("cloned repo missing README.md: %v", err)
	}
}

func TestHashSpec(t *testing.T) {
	a := hashSpec("https://github.com/foo/bar")
	b := hashSpec("https://github.com/foo/bar")
	c := hashSpec("https://github.com/other/repo")
	if a != b {
		t.Error("same input should produce same hash")
	}
	if a == c {
		t.Error("different input should produce different hash")
	}
	if len(a) != 16 {
		t.Errorf("hash length = %d, want 16", len(a))
	}
}
