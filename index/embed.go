package index

import (
	"hash/fnv"
	"math"
)

// Embedder turns a token list into a dense, L2-normalized vector.
type Embedder interface {
	Embed(tokens []string) []float32
	Dim() int
}

// HashEmbedder is a pure-Go static embedder. It hashes each token and its
// character trigrams into a fixed-dimension vector (the "hashing trick"),
// giving a dense representation where morphologically similar tokens share
// dimensions. It is the lightweight default; the Embedder interface lets a
// real Model2Vec backend drop in later without touching the index.
type HashEmbedder struct{ dim int }

func NewHashEmbedder(dim int) *HashEmbedder {
	if dim <= 0 {
		dim = 256
	}
	return &HashEmbedder{dim: dim}
}

func (h *HashEmbedder) Dim() int { return h.dim }

func (h *HashEmbedder) Embed(tokens []string) []float32 {
	v := make([]float32, h.dim)
	for _, t := range tokens {
		h.add(v, t, 1.0)
		for _, tri := range trigrams(t) {
			h.add(v, tri, 0.5)
		}
	}
	norm := float32(0)
	for _, x := range v {
		norm += x * x
	}
	if norm > 0 {
		inv := float32(1.0 / math.Sqrt(float64(norm)))
		for i := range v {
			v[i] *= inv
		}
	}
	return v
}

func (h *HashEmbedder) add(v []float32, s string, w float32) {
	hh := fnv.New32a()
	_, _ = hh.Write([]byte(s))
	sum := hh.Sum32()
	idx := int(sum % uint32(h.dim))
	// sign hashing reduces collisions biasing the vector in one direction
	if sum&0x80000000 != 0 {
		v[idx] -= w
	} else {
		v[idx] += w
	}
}

func trigrams(s string) []string {
	r := []rune(s)
	if len(r) < 3 {
		return nil
	}
	out := make([]string, 0, len(r)-2)
	for i := 0; i+3 <= len(r); i++ {
		out = append(out, string(r[i:i+3]))
	}
	return out
}

// cosine returns the dot product of two L2-normalized vectors.
func cosine(a, b []float32) float64 {
	var dot float32
	for i := range a {
		dot += a[i] * b[i]
	}
	return float64(dot)
}
