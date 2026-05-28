package mcpserver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// setupRepo creates a temp git repo with a Go file for testing.
func setupRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	// Write a Go file with identifiable content.
	code := `package main

import "net/http"

// retryRequest retries an HTTP request up to n times.
func retryRequest(client *http.Client, req *http.Request, n int) (*http.Response, error) {
	var resp *http.Response
	var err error
	for i := 0; i < n; i++ {
		resp, err = client.Do(req)
		if err == nil {
			return resp, nil
		}
	}
	return resp, err
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(code), 0o644); err != nil {
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
	return dir
}

func callTool(s *Server, name string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	st := s.MCP().GetTool(name)
	if st == nil {
		return nil, fmt.Errorf("tool %q not found", name)
	}
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	return st.Handler(context.Background(), req)
}

func TestHandleSearch(t *testing.T) {
	repo := setupRepo(t)
	s := New("test")

	result, err := callTool(s, "search", map[string]interface{}{
		"query": "retry http request on failure",
		"repo":  repo,
	})
	if err != nil {
		t.Fatalf("handleSearch: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %v", result.Content)
	}
}

func TestHandleSearchMissingQuery(t *testing.T) {
	s := New("test")
	result, err := callTool(s, "search", map[string]interface{}{
		"repo": ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing query")
	}
}

func TestHandleFindRelated(t *testing.T) {
	repo := setupRepo(t)
	s := New("test")

	result, err := callTool(s, "find_related", map[string]interface{}{
		"file": "main.go",
		"line": 6.0, // JSON numbers are float64
		"repo": repo,
	})
	if err != nil {
		t.Fatalf("handleFindRelated: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %v", result.Content)
	}
}

func TestHandleFindRelatedMissingFile(t *testing.T) {
	s := New("test")
	result, err := callTool(s, "find_related", map[string]interface{}{
		"line": 1.0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing file")
	}
}
