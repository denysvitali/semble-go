package index

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// alwaysSkip are well-known non-source directories that are skipped regardless
// of ignore files.
var alwaysSkip = map[string]bool{
	".git": true, "node_modules": true, ".venv": true, "venv": true,
	"__pycache__": true, "vendor": true, "dist": true, "build": true,
	"target": true, ".next": true, ".idea": true, ".cache": true,
	".mypy_cache": true, ".pytest_cache": true, ".gradle": true,
}

// codeExts is the set of extensions indexed as code.
var codeExts = map[string]bool{
	".go": true, ".py": true, ".js": true, ".jsx": true, ".ts": true, ".tsx": true,
	".java": true, ".c": true, ".h": true, ".cpp": true, ".cc": true, ".hpp": true,
	".cs": true, ".rb": true, ".rs": true, ".php": true, ".swift": true, ".kt": true,
	".scala": true, ".sh": true, ".bash": true, ".lua": true, ".r": true, ".m": true,
	".sql": true, ".dart": true, ".ex": true, ".exs": true, ".clj": true, ".hs": true,
	".vue": true, ".svelte": true, ".proto": true, ".gradle": true,
}

type fileInfo struct {
	rel   string
	mtime int64
	size  int64
}

type ignoreList struct {
	deny  []string
	allow []string // "!" force-include patterns
}

func loadIgnore(root string) *ignoreList {
	il := &ignoreList{}
	for _, name := range []string{".gitignore", ".sembleignore"} {
		f, err := os.Open(filepath.Join(root, name))
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "!") {
				il.allow = append(il.allow, strings.TrimPrefix(line, "!"))
			} else {
				il.deny = append(il.deny, strings.TrimRight(line, "/"))
			}
		}
		f.Close()
	}
	return il
}

func (il *ignoreList) ignored(rel string) bool {
	base := filepath.Base(rel)
	for _, p := range il.allow {
		if matchPattern(p, rel, base) {
			return false
		}
	}
	for _, p := range il.deny {
		if matchPattern(p, rel, base) {
			return true
		}
	}
	return false
}

func matchPattern(pat, rel, base string) bool {
	pat = strings.TrimPrefix(pat, "/")
	if ok, _ := filepath.Match(pat, base); ok {
		return true
	}
	if ok, _ := filepath.Match(pat, rel); ok {
		return true
	}
	// directory prefix match (e.g. "build" matches "build/foo.go")
	if strings.HasPrefix(rel, pat+"/") {
		return true
	}
	return false
}

// walk collects indexable files under root, honoring ignore rules and the
// requested kinds (code/docs/config). force-included extensions via
// .sembleignore "!" rules are always considered.
func walk(root string, kinds map[string]bool, il *ignoreList) ([]fileInfo, error) {
	var files []fileInfo
	err := filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil || rel == "." {
			return nil
		}
		if fi.IsDir() {
			if alwaysSkip[fi.Name()] || il.ignored(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if il.ignored(rel) {
			return nil
		}
		forced := false
		for _, p := range il.allow {
			if matchPattern(p, rel, filepath.Base(rel)) {
				forced = true
				break
			}
		}
		if !forced && !indexable(rel, kinds) {
			return nil
		}
		files = append(files, fileInfo{rel: rel, mtime: fi.ModTime().Unix(), size: fi.Size()})
		return nil
	})
	return files, err
}

func indexable(rel string, kinds map[string]bool) bool {
	ext := strings.ToLower(filepath.Ext(rel))
	switch {
	case kinds["code"] && codeExts[ext]:
		return true
	case kinds["docs"] && docExts[ext]:
		return true
	case kinds["config"] && configExts[ext]:
		return true
	}
	return false
}
