package index

import (
	"encoding/json"
	"testing"
)

func TestFormatResultsJSON(t *testing.T) {
	results := []Result{
		{Chunk: Chunk{File: "a.go", Start: 1, End: 10, Kind: "code", Text: "package a"}, Score: 0.5},
		{Chunk: Chunk{File: "b.md", Start: 3, End: 7, Kind: "docs", Text: "# title"}, Score: 0.25},
	}
	out, err := FormatResultsJSON(results)
	if err != nil {
		t.Fatalf("FormatResultsJSON: %v", err)
	}
	var got []struct {
		File  string  `json:"file"`
		Start int     `json:"start"`
		End   int     `json:"end"`
		Kind  string  `json:"kind"`
		Score float64 `json:"score"`
		Text  string  `json:"text"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 results, got %d", len(got))
	}
	if got[0].File != "a.go" || got[0].Start != 1 || got[0].End != 10 ||
		got[0].Kind != "code" || got[0].Score != 0.5 || got[0].Text != "package a" {
		t.Errorf("unexpected first result: %+v", got[0])
	}
}

func TestFormatResultsJSONEmpty(t *testing.T) {
	out, err := FormatResultsJSON(nil)
	if err != nil {
		t.Fatalf("FormatResultsJSON: %v", err)
	}
	if out != "[]" {
		t.Errorf("want empty array, got %q", out)
	}
}

func TestFormatSavingsJSON(t *testing.T) {
	s := Savings{Files: 3, Chunks: 12, CorpusTokens: 4000, SearchTokens: 200, Ratio: 20}
	out, err := FormatSavingsJSON(s)
	if err != nil {
		t.Fatalf("FormatSavingsJSON: %v", err)
	}
	var got struct {
		Files        int     `json:"files"`
		Chunks       int     `json:"chunks"`
		CorpusTokens int     `json:"corpus_tokens"`
		SearchTokens int     `json:"search_tokens"`
		Ratio        float64 `json:"ratio"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if got.Files != 3 || got.Chunks != 12 || got.CorpusTokens != 4000 ||
		got.SearchTokens != 200 || got.Ratio != 20 {
		t.Errorf("unexpected savings: %+v", got)
	}
}
