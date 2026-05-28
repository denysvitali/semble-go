package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("SEMBLE_CACHE_DIR", filepath.Join(dir, ".cache"))
	files := map[string]string{
		"http.go": `package net

// retryRequest re-sends an HTTP request on transient failure.
func retryRequest(client *Client, req *Request) (*Response, error) {
	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err := client.Do(req)
		if err == nil {
			return resp, nil
		}
		time.Sleep(backoff(attempt))
	}
	return nil, errTooManyRetries
}
`,
		"auth.go": `package auth

// LoginUser validates credentials and returns a session token.
func LoginUser(username, password string) (string, error) {
	if !checkPassword(username, password) {
		return "", errInvalidCredentials
	}
	return newSessionToken(username), nil
}
`,
		"readme.md":   "# Project\n\nThis project does retries and authentication.\n",
		"vendor/x.go": "package x\nfunc Junk() {}\n",
	}
	for rel, body := range files {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestSearchFindsRelevantChunk(t *testing.T) {
	dir := setupRepo(t)
	idx, err := Open(dir, map[string]bool{"code": true})
	if err != nil {
		t.Fatal(err)
	}
	results := idx.Search("retry http request on failure", 5)
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	if results[0].Chunk.File != "http.go" {
		t.Fatalf("expected http.go first, got %s", results[0].Chunk.File)
	}
	for _, r := range results {
		if strings.HasPrefix(r.Chunk.File, "vendor/") {
			t.Fatalf("vendor/ should be skipped, got %s", r.Chunk.File)
		}
	}
}

func TestCacheRoundTrip(t *testing.T) {
	dir := setupRepo(t)
	if _, err := Open(dir, map[string]bool{"code": true}); err != nil {
		t.Fatal(err)
	}
	idx, err := Open(dir, map[string]bool{"code": true}) // second open hits cache
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Chunks) == 0 || len(idx.vecs) != len(idx.Chunks) {
		t.Fatalf("cache load produced inconsistent index: %d chunks, %d vecs", len(idx.Chunks), len(idx.vecs))
	}
}

func TestFindRelated(t *testing.T) {
	dir := setupRepo(t)
	idx, err := Open(dir, AllKinds)
	if err != nil {
		t.Fatal(err)
	}
	results, err := idx.FindRelated("http.go", 4, 5)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.Chunk.File == "http.go" {
			t.Fatal("find-related must exclude the source file")
		}
	}
}

func TestTokenizeSplitsIdentifiers(t *testing.T) {
	got := Tokenize("retryRequest maxRetries HTTP2Client")
	want := map[string]bool{"retryrequest": false, "retry": false, "request": false, "max": false, "retries": false, "http": false, "client": false}
	for _, tok := range got {
		if _, ok := want[tok]; ok {
			want[tok] = true
		}
	}
	for tok, seen := range want {
		if !seen {
			t.Errorf("expected token %q in %v", tok, got)
		}
	}
}
