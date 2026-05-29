package index

import "encoding/json"

// jsonResult is the clean, stable shape exposed by --json, decoupled from the
// internal Chunk/Result types.
type jsonResult struct {
	File  string  `json:"file"`
	Start int     `json:"start"`
	End   int     `json:"end"`
	Kind  string  `json:"kind"`
	Score float64 `json:"score"`
	Text  string  `json:"text"`
}

// FormatResultsJSON renders results as pretty-printed JSON. Empty results yield
// an empty array rather than null.
func FormatResultsJSON(results []Result) (string, error) {
	out := make([]jsonResult, 0, len(results))
	for _, r := range results {
		out = append(out, jsonResult{
			File:  r.Chunk.File,
			Start: r.Chunk.Start,
			End:   r.Chunk.End,
			Kind:  r.Chunk.Kind,
			Score: r.Score,
			Text:  r.Chunk.Text,
		})
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// FormatSavingsJSON renders a token-savings estimate as pretty-printed JSON.
func FormatSavingsJSON(s Savings) (string, error) {
	b, err := json.MarshalIndent(struct {
		Files        int     `json:"files"`
		Chunks       int     `json:"chunks"`
		CorpusTokens int     `json:"corpus_tokens"`
		SearchTokens int     `json:"search_tokens"`
		Ratio        float64 `json:"ratio"`
	}{
		Files:        s.Files,
		Chunks:       s.Chunks,
		CorpusTokens: s.CorpusTokens,
		SearchTokens: s.SearchTokens,
		Ratio:        s.Ratio,
	}, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
