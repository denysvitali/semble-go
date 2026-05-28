package index

import "math"

type posting struct {
	doc int
	tf  int
}

// BM25 is a compact Okapi BM25 lexical index over tokenized documents.
type BM25 struct {
	postings map[string][]posting
	idf      map[string]float64
	docLen   []int
	avgdl    float64
	k1, b    float64
}

func newBM25(docTokens [][]string) *BM25 {
	b := &BM25{
		postings: map[string][]posting{},
		idf:      map[string]float64{},
		docLen:   make([]int, len(docTokens)),
		k1:       1.5,
		b:        0.75,
	}
	var total int
	for d, toks := range docTokens {
		b.docLen[d] = len(toks)
		total += len(toks)
		counts := map[string]int{}
		for _, t := range toks {
			counts[t]++
		}
		for t, tf := range counts {
			b.postings[t] = append(b.postings[t], posting{doc: d, tf: tf})
		}
	}
	if n := len(docTokens); n > 0 {
		b.avgdl = float64(total) / float64(n)
	}
	n := float64(len(docTokens))
	for t, pl := range b.postings {
		df := float64(len(pl))
		b.idf[t] = math.Log(1 + (n-df+0.5)/(df+0.5))
	}
	return b
}

// Score returns a doc->score map for the query terms (only docs with score > 0).
func (b *BM25) Score(query []string) map[int]float64 {
	scores := map[int]float64{}
	seen := map[string]bool{}
	for _, t := range query {
		if seen[t] {
			continue
		}
		seen[t] = true
		pl, ok := b.postings[t]
		if !ok {
			continue
		}
		idf := b.idf[t]
		for _, p := range pl {
			tf := float64(p.tf)
			dl := float64(b.docLen[p.doc])
			denom := tf + b.k1*(1-b.b+b.b*dl/b.avgdl)
			scores[p.doc] += idf * (tf * (b.k1 + 1)) / denom
		}
	}
	return scores
}
