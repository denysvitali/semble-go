package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/denysvitali/semble-go/index"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server exposing semantic code search.
type Server struct {
	srv *server.MCPServer
}

func New(version string) *Server {
	s := server.NewMCPServer(
		"semble-go",
		version,
		server.WithToolCapabilities(true),
	)
	srv := &Server{srv: s}
	srv.register()
	return srv
}

func (s *Server) MCP() *server.MCPServer { return s.srv }

func (s *Server) register() {
	s.srv.AddTool(mcp.NewTool("search",
		mcp.WithDescription("Semantic + lexical code search over a local repository. Returns only the most relevant code chunks (path:lines + snippet), not whole files — typically ~98% fewer tokens than grep+read."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithString("query", mcp.Required(),
			mcp.Description("Natural-language or code query (e.g. 'where are http requests retried')"),
		),
		mcp.WithString("repo",
			mcp.Description("Path to the local repository to search (default: current directory)"),
			mcp.DefaultString("."),
		),
		mcp.WithNumber("top_k",
			mcp.Description("Maximum number of chunks to return (default: 10)"),
			mcp.DefaultNumber(10),
		),
		mcp.WithString("content",
			mcp.Description("Which files to index: code (default), docs, config, or all"),
			mcp.DefaultString("code"),
		),
	), s.handleSearch)

	s.srv.AddTool(mcp.NewTool("find_related",
		mcp.WithDescription("Find code chunks semantically similar to a given file:line location, excluding the source file itself."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithString("file", mcp.Required(),
			mcp.Description("File path (relative to repo) to anchor the search"),
		),
		mcp.WithNumber("line", mcp.Required(),
			mcp.Description("1-based line number within the file"),
		),
		mcp.WithString("repo",
			mcp.Description("Path to the local repository (default: current directory)"),
			mcp.DefaultString("."),
		),
		mcp.WithNumber("top_k",
			mcp.Description("Maximum number of chunks to return (default: 10)"),
			mcp.DefaultNumber(10),
		),
	), s.handleFindRelated)
}

func (s *Server) handleSearch(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	query := argStr(args, "query", "")
	if strings.TrimSpace(query) == "" {
		return mcp.NewToolResultError("query is required"), nil
	}
	repo := argStr(args, "repo", ".")
	topK := argInt(args, "top_k", 10)
	content := argStr(args, "content", "code")

	idx, err := index.Open(repo, index.KindsFromContent(content))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to index %q: %v", repo, err)), nil
	}
	results := idx.Search(query, topK)
	return mcp.NewToolResultText(index.FormatResults(results)), nil
}

func (s *Server) handleFindRelated(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	file := argStr(args, "file", "")
	if strings.TrimSpace(file) == "" {
		return mcp.NewToolResultError("file is required"), nil
	}
	line := argInt(args, "line", 0)
	repo := argStr(args, "repo", ".")
	topK := argInt(args, "top_k", 10)

	idx, err := index.Open(repo, index.AllKinds)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to index %q: %v", repo, err)), nil
	}
	results, err := idx.FindRelated(file, line, topK)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(index.FormatResults(results)), nil
}

func argStr(args map[string]interface{}, key, def string) string {
	if v, ok := args[key].(string); ok && v != "" {
		return v
	}
	return def
}

func argInt(args map[string]interface{}, key string, def int) int {
	if v, ok := args[key].(float64); ok {
		return int(v)
	}
	return def
}
