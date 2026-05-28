package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/denysvitali/semble-go/index"
	"github.com/denysvitali/semble-go/mcpserver"
	"github.com/denysvitali/semble-go/model2vec"

	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

var version = "dev"

var (
	flagTopK    int
	flagContent string
	flagAgent   string
	flagModel   string
)

var rootCmd = &cobra.Command{
	Use:   "semble",
	Short: "Token-efficient semantic code search",
	Long: `Semble indexes a repository into code-aware chunks and answers queries by
fusing BM25 lexical retrieval with semantic embeddings (Reciprocal Rank Fusion),
returning only the most relevant snippets instead of whole files.`,
	SilenceUsage: true,
}

func openIndex(path string, kinds map[string]bool) (*index.Index, error) {
	if flagModel != "" {
		m, err := model2vec.Load(flagModel)
		if err != nil {
			return nil, fmt.Errorf("load model %q: %w", flagModel, err)
		}
		return index.OpenWith(path, kinds, m)
	}
	return index.Open(path, kinds)
}

var searchCmd = &cobra.Command{
	Use:   "search <query> [path]",
	Short: "Search a repository for relevant code chunks",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) == 2 {
			path = args[1]
		}
		idx, err := openIndex(path, index.KindsFromContent(flagContent))
		if err != nil {
			return err
		}
		fmt.Println(index.FormatResults(idx.Search(args[0], flagTopK)))
		return nil
	},
}

var findRelatedCmd = &cobra.Command{
	Use:   "find-related <file> <line> [path]",
	Short: "Find chunks semantically similar to file:line",
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		line, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid line number %q: %w", args[1], err)
		}
		path := "."
		if len(args) == 3 {
			path = args[2]
		}
		idx, err := openIndex(path, index.AllKinds)
		if err != nil {
			return err
		}
		results, err := idx.FindRelated(args[0], line, flagTopK)
		if err != nil {
			return err
		}
		fmt.Println(index.FormatResults(results))
		return nil
	},
}

var savingsCmd = &cobra.Command{
	Use:   "savings [path]",
	Short: "Estimate token savings vs reading the whole corpus",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) == 1 {
			path = args[0]
		}
		idx, err := openIndex(path, index.KindsFromContent(flagContent))
		if err != nil {
			return err
		}
		s := idx.Estimate(flagTopK)
		fmt.Printf("files:         %d\n", s.Files)
		fmt.Printf("chunks:        %d\n", s.Chunks)
		fmt.Printf("corpus tokens: ~%d\n", s.CorpusTokens)
		fmt.Printf("search tokens: ~%d (top-%d)\n", s.SearchTokens, flagTopK)
		fmt.Printf("savings:       %.1fx fewer tokens\n", s.Ratio)
		return nil
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run as an MCP server over stdio",
	RunE: func(cmd *cobra.Command, args []string) error {
		return server.ServeStdio(mcpserver.New(version).MCP())
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Print MCP server configuration for an agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		bin, err := os.Executable()
		if err != nil || bin == "" {
			bin = "semble"
		}
		fmt.Print(mcpConfig(flagAgent, bin))
		return nil
	},
}

func init() {
	searchCmd.Flags().IntVar(&flagTopK, "top-k", 10, "maximum chunks to return")
	searchCmd.Flags().StringVar(&flagContent, "content", "code", "files to index: code|docs|config|all")
	searchCmd.Flags().StringVar(&flagModel, "model", "", "path to Model2Vec model directory")
	findRelatedCmd.Flags().IntVar(&flagTopK, "top-k", 10, "maximum chunks to return")
	findRelatedCmd.Flags().StringVar(&flagModel, "model", "", "path to Model2Vec model directory")
	savingsCmd.Flags().IntVar(&flagTopK, "top-k", 10, "result size used for the estimate")
	savingsCmd.Flags().StringVar(&flagContent, "content", "code", "files to index: code|docs|config|all")
	savingsCmd.Flags().StringVar(&flagModel, "model", "", "path to Model2Vec model directory")
	initCmd.Flags().StringVar(&flagAgent, "agent", "claude", "agent: claude|cursor|codex|generic")

	rootCmd.AddCommand(searchCmd, findRelatedCmd, savingsCmd, serveCmd, initCmd)
}

// mcpConfig prints the MCP server stanza. The shape is shared across Claude
// Code, Cursor and Codex; only the destination file differs per agent.
func mcpConfig(agent, bin string) string {
	dest := map[string]string{
		"claude": ".mcp.json",
		"cursor": ".cursor/mcp.json",
		"codex":  "~/.codex/config (mcp_servers)",
	}[agent]
	if dest == "" {
		dest = ".mcp.json"
	}
	return fmt.Sprintf("# add to %s\n{\n  \"mcpServers\": {\n    \"semble\": {\n      \"command\": %q,\n      \"args\": [\"serve\"]\n    }\n  }\n}\n", dest, bin)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
