package index

import "sort"

// rrfK is the standard Reciprocal Rank Fusion constant.
const rrfK = 60.0

// fuse combines several ranked lists of document ids into a single ranking
// using Reciprocal Rank Fusion: a doc at rank r (0-based) contributes
// 1/(k+r+1) per list. Returns doc ids sorted by fused score, descending.
func fuse(rankings ...[]int) []scored {
	agg := map[int]float64{}
	for _, ranking := range rankings {
		for r, doc := range ranking {
			agg[doc] += 1.0 / (rrfK + float64(r) + 1.0)
		}
	}
	out := make([]scored, 0, len(agg))
	for doc, s := range agg {
		out = append(out, scored{doc: doc, score: s})
	}
	sortScored(out)
	return out
}

type scored struct {
	doc   int
	score float64
}

// sortScored orders chunks by score descending, breaking ties by ascending
// doc id so rankings are deterministic.
func sortScored(s []scored) {
	sort.Slice(s, func(i, j int) bool {
		if s[i].score != s[j].score {
			return s[i].score > s[j].score
		}
		return s[i].doc < s[j].doc
	})
}

// topDocs sorts a doc->score map descending and returns up to poolSize doc ids.
func topDocs(scores map[int]float64) []int {
	s := make([]scored, 0, len(scores))
	for d, v := range scores {
		s = append(s, scored{doc: d, score: v})
	}
	sortScored(s)
	if len(s) > poolSize {
		s = s[:poolSize]
	}
	out := make([]int, len(s))
	for i, x := range s {
		out[i] = x.doc
	}
	return out
}
