package index

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

var urlRe = regexp.MustCompile(`^(https?|git|ssh|file)://|^git@`)

// IsRemote reports whether spec looks like a git URL rather than a local path.
func IsRemote(spec string) bool {
	return urlRe.MatchString(spec)
}

// ResolveRepo turns a repo spec into a local directory. Local paths are
// returned unchanged; git URLs are cloned (shallow) into the cache and reused
// — re-fetched on subsequent calls so the working tree tracks the remote.
func ResolveRepo(spec string) (string, error) {
	if !IsRemote(spec) {
		return spec, nil
	}
	dest := filepath.Join(cacheDir(), "repos", hashSpec(spec))
	timeout := cloneTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if _, err := os.Stat(filepath.Join(dest, ".git")); err == nil {
		if err := gitFetch(ctx, dest); err != nil {
			return "", err
		}
		return dest, nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	_ = os.RemoveAll(dest)
	if err := runGit(ctx, "", "clone", "--depth", "1", spec, dest); err != nil {
		return "", fmt.Errorf("clone %s: %w", spec, err)
	}
	return dest, nil
}

func gitFetch(ctx context.Context, dest string) error {
	if err := runGit(ctx, dest, "fetch", "--depth", "1", "origin"); err != nil {
		return fmt.Errorf("fetch %s: %w", dest, err)
	}
	// Move the working tree to whatever the remote HEAD now points at.
	if err := runGit(ctx, dest, "reset", "--hard", "FETCH_HEAD"); err != nil {
		return fmt.Errorf("reset %s: %w", dest, err)
	}
	return nil
}

func runGit(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	return nil
}

func hashSpec(spec string) string {
	sum := sha256.Sum256([]byte(spec))
	return hex.EncodeToString(sum[:])[:16]
}

func cloneTimeout() time.Duration {
	if v := os.Getenv("SEMBLE_CLONE_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 60 * time.Second
}
