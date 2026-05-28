package index

import (
	"fmt"
	"strings"
)

// FormatResults renders results in a compact, token-efficient form:
// a header line per hit followed by the snippet, separated by a thin rule.
func FormatResults(results []Result) string {
	if len(results) == 0 {
		return "(no results)"
	}
	var sb strings.Builder
	for i, r := range results {
		if i > 0 {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "%s:%d-%d  [%s] score=%.4f\n", r.Chunk.File, r.Chunk.Start, r.Chunk.End, r.Chunk.Kind, r.Score)
		sb.WriteString(strings.TrimRight(r.Chunk.Text, "\n"))
		sb.WriteString("\n")
	}
	return sb.String()
}
