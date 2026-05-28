package model2vec

import (
	"fmt"
	"math"
	"path/filepath"
)

// StaticModel implements index.Embedder using a Model2Vec static embedding
// model stored as safetensors + tokenizer.json.
type StaticModel struct {
	tok     *WordPiece
	embeds  []float32 // flattened [vocab_size * dim]
	dim     int
	vocabSz int
}

// Load loads a Model2Vec model from a local directory containing
// model.safetensors and tokenizer.json.
func Load(dir string) (*StaticModel, error) {
	st, err := LoadSafetensors(filepath.Join(dir, "model.safetensors"))
	if err != nil {
		return nil, fmt.Errorf("load safetensors: %w", err)
	}
	tok, err := LoadTokenizer(filepath.Join(dir, "tokenizer.json"))
	if err != nil {
		return nil, fmt.Errorf("load tokenizer: %w", err)
	}
	embedTensor, ok := st.Tensors["embeddings"]
	if !ok {
		return nil, fmt.Errorf("safetensors: missing 'embeddings' tensor")
	}
	shape := st.Shapes["embeddings"]
	if len(shape) != 2 {
		return nil, fmt.Errorf("embeddings tensor must be 2D, got %dD", len(shape))
	}
	vocabSz, dim := shape[0], shape[1]
	if len(embedTensor) != vocabSz*dim {
		return nil, fmt.Errorf("embeddings size mismatch: shape %v but %d elements", shape, len(embedTensor))
	}
	return &StaticModel{
		tok:     tok,
		embeds:  embedTensor,
		dim:     dim,
		vocabSz: vocabSz,
	}, nil
}

func (m *StaticModel) Dim() int   { return m.dim }
func (m *StaticModel) ID() string { return fmt.Sprintf("model2vec-%d", m.dim) }

func (m *StaticModel) Embed(text string) []float32 {
	ids := m.tok.Tokenize(text)
	if len(ids) == 0 {
		v := make([]float32, m.dim)
		return v
	}
	v := make([]float32, m.dim)
	count := 0
	for _, id := range ids {
		if int(id) >= m.vocabSz {
			continue
		}
		off := int(id) * m.dim
		for i := 0; i < m.dim; i++ {
			v[i] += m.embeds[off+i]
		}
		count++
	}
	if count > 0 {
		inv := float32(1.0 / float64(count))
		for i := range v {
			v[i] *= inv
		}
	}
	// L2 normalize
	var norm float32
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
