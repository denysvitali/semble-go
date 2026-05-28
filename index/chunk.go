package index

import (
	"path/filepath"
	"strings"
)

// Chunk is a contiguous, searchable slice of a file.
type Chunk struct {
	File  string // path relative to index root
	Start int    // 1-based first line
	End   int    // 1-based last line (inclusive)
	Text  string
	Kind  string // "code", "docs" or "config"
}

const (
	chunkMaxLines = 50
	chunkMinLines = 8
)

// chunkFile splits a file into code-aware-ish chunks. We avoid CGO/tree-sitter
// and instead break on blank lines once a minimum size is reached, with a hard
// cap. This keeps chunks aligned to logical blocks (functions, blocks, paragraphs)
// without a language grammar.
func chunkFile(rel string, data []byte) []Chunk {
	kind := kindForExt(rel)
	lines := strings.Split(string(data), "\n")
	var chunks []Chunk
	start := 0
	for i := 0; i < len(lines); i++ {
		n := i - start + 1
		blank := strings.TrimSpace(lines[i]) == ""
		if n >= chunkMaxLines || (n >= chunkMinLines && blank) {
			if c, ok := makeChunk(rel, kind, lines, start, i); ok {
				chunks = append(chunks, c)
			}
			start = i + 1
		}
	}
	if start < len(lines) {
		if c, ok := makeChunk(rel, kind, lines, start, len(lines)-1); ok {
			chunks = append(chunks, c)
		}
	}
	return chunks
}

func makeChunk(rel, kind string, lines []string, start, end int) (Chunk, bool) {
	text := strings.Join(lines[start:end+1], "\n")
	if strings.TrimSpace(text) == "" {
		return Chunk{}, false
	}
	return Chunk{
		File:  rel,
		Start: start + 1,
		End:   end + 1,
		Text:  text,
		Kind:  kind,
	}, true
}

var docExts = map[string]bool{".md": true, ".markdown": true, ".rst": true, ".txt": true, ".adoc": true}
var configExts = map[string]bool{
	".yaml": true, ".yml": true, ".toml": true, ".json": true, ".ini": true,
	".cfg": true, ".conf": true, ".env": true, ".properties": true,
}

func kindForExt(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch {
	case docExts[ext]:
		return "docs"
	case configExts[ext]:
		return "config"
	default:
		return "code"
	}
}
