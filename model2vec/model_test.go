package model2vec

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// createTestModel writes minimal synthetic safetensors + tokenizer.json fixtures
// and returns a loaded StaticModel.
func createTestModel(t *testing.T, dim, vocabSize int) *StaticModel {
	t.Helper()
	dir := t.TempDir()

	// Build embeddings tensor: vocabSize x dim, identity-ish pattern.
	// Token i gets vector with 1.0 at position (i % dim).
	data := make([]byte, vocabSize*dim*4)
	for i := 0; i < vocabSize; i++ {
		v := float32(0)
		if dim > 0 {
			v = 1.0
		}
		idx := i % dim
		binary.LittleEndian.PutUint32(data[(i*dim+idx)*4:], math.Float32bits(v))
	}

	// safetensors: 8-byte header len + JSON header + data
	header := map[string]interface{}{
		"embeddings": map[string]interface{}{
			"dtype":        "F32",
			"shape":        []int{vocabSize, dim},
			"data_offsets": []int64{0, int64(len(data))},
		},
	}
	hdrJSON, _ := json.Marshal(header)
	hdrLen := make([]byte, 8)
	binary.LittleEndian.PutUint64(hdrLen, uint64(len(hdrJSON)))

	stFile := filepath.Join(dir, "model.safetensors")
	var stData []byte
	stData = append(stData, hdrLen...)
	stData = append(stData, hdrJSON...)
	stData = append(stData, data...)
	if err := os.WriteFile(stFile, stData, 0o644); err != nil {
		t.Fatal(err)
	}

	// tokenizer.json: simple vocab mapping words to IDs
	vocab := map[string]int32{
		"[UNK]": 0,
		"[CLS]": 1,
		"[SEP]": 2,
		"hello": 3,
		"world": 4,
		"go":    5,
		"code":  6,
		"the":   7,
		"a":     8,
		"is":    9,
	}
	for i := 10; i < vocabSize; i++ {
		vocab[tokenFromID(i)] = int32(i)
	}
	tokCfg := map[string]interface{}{
		"model": map[string]interface{}{
			"vocab":                     vocab,
			"unk_token":                 "[UNK]",
			"continuing_subword_prefix": "##",
			"max_input_chars_per_word":  100,
		},
	}
	tokJSON, _ := json.Marshal(tokCfg)
	if err := os.WriteFile(filepath.Join(dir, "tokenizer.json"), tokJSON, 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return m
}

func tokenFromID(id int) string {
	// Simple deterministic token names for IDs >= 10.
	return "tok" + string(rune('a'+id%26)) + string(rune('a'+(id/26)%26))
}

func TestLoadAndEmbed(t *testing.T) {
	dim := 32
	m := createTestModel(t, dim, 20)
	if m.Dim() != dim {
		t.Fatalf("dim = %d, want %d", m.Dim(), dim)
	}
	if m.ID() != "model2vec-32" {
		t.Fatalf("id = %q, want model2vec-32", m.ID())
	}

	vec := m.Embed("hello world")
	if len(vec) != dim {
		t.Fatalf("vec len = %d, want %d", len(vec), dim)
	}
	// Verify L2-normalized.
	var norm float32
	for _, x := range vec {
		norm += x * x
	}
	if math.Abs(float64(norm)-1.0) > 0.01 {
		t.Fatalf("vec not L2-normalized: norm^2 = %f", norm)
	}
}

func TestEmbedEmpty(t *testing.T) {
	m := createTestModel(t, 16, 10)
	vec := m.Embed("")
	if len(vec) != 16 {
		t.Fatalf("empty embed len = %d", len(vec))
	}
	for _, x := range vec {
		if x != 0 {
			t.Fatalf("empty embed should be zero vector, got %f", x)
		}
	}
}

func TestEmbedUnknownTokens(t *testing.T) {
	m := createTestModel(t, 16, 10)
	// "xyz" is not in vocab, should fall back to [UNK].
	vec := m.Embed("xyz")
	if len(vec) != 16 {
		t.Fatalf("vec len = %d", len(vec))
	}
	var norm float32
	for _, x := range vec {
		norm += x * x
	}
	if norm == 0 {
		t.Fatal("expected non-zero vector for unknown token")
	}
}

func TestEmbedDeterministic(t *testing.T) {
	m := createTestModel(t, 32, 20)
	v1 := m.Embed("hello code")
	v2 := m.Embed("hello code")
	for i := range v1 {
		if v1[i] != v2[i] {
			t.Fatalf("non-deterministic at index %d: %f vs %f", i, v1[i], v2[i])
		}
	}
}

func TestSafetensorsF16(t *testing.T) {
	dir := t.TempDir()
	dim := 4
	vocab := 2

	// F16 data: 2 tokens x 4 dims = 8 float16 values = 16 bytes.
	data := make([]byte, 16)
	vals := []float32{1.0, 0.5, -1.0, 0.25, 0.0, 2.0, -0.5, 0.75}
	for i, v := range vals {
		binary.LittleEndian.PutUint16(data[i*2:], float32ToF16(v))
	}
	header := map[string]interface{}{
		"embeddings": map[string]interface{}{
			"dtype":        "F16",
			"shape":        []int{vocab, dim},
			"data_offsets": []int64{0, int64(len(data))},
		},
	}
	hdrJSON, _ := json.Marshal(header)
	hdrLen := make([]byte, 8)
	binary.LittleEndian.PutUint64(hdrLen, uint64(len(hdrJSON)))
	var stData []byte
	stData = append(stData, hdrLen...)
	stData = append(stData, hdrJSON...)
	stData = append(stData, data...)
	if err := os.WriteFile(filepath.Join(dir, "model.safetensors"), stData, 0o644); err != nil {
		t.Fatal(err)
	}

	tokCfg := map[string]interface{}{
		"model": map[string]interface{}{
			"vocab": map[string]int32{"a": 0, "b": 1},
		},
	}
	tokJSON, _ := json.Marshal(tokCfg)
	if err := os.WriteFile(filepath.Join(dir, "tokenizer.json"), tokJSON, 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	vec := m.Embed("a")
	if len(vec) != dim {
		t.Fatalf("vec len = %d, want %d", len(vec), dim)
	}
}

func float32ToF16(f float32) uint16 {
	bits := math.Float32bits(f)
	sign := (bits >> 16) & 0x8000
	exp := int32((bits>>23)&0xFF) - 127 + 15
	frac := (bits >> 13) & 0x3FF
	if exp <= 0 {
		return uint16(sign)
	}
	if exp >= 0x1F {
		return uint16(sign | 0x7C00 | frac)
	}
	return uint16(sign | uint32(exp)<<10 | frac)
}

func TestSafetensorsMissingTensor(t *testing.T) {
	dir := t.TempDir()
	header := map[string]interface{}{
		"other": map[string]interface{}{
			"dtype": "F32", "shape": []int{1, 1},
			"data_offsets": []int64{0, 4},
		},
	}
	hdrJSON, _ := json.Marshal(header)
	hdrLen := make([]byte, 8)
	binary.LittleEndian.PutUint64(hdrLen, uint64(len(hdrJSON)))
	var stData []byte
	stData = append(stData, hdrLen...)
	stData = append(stData, hdrJSON...)
	stData = append(stData, make([]byte, 4)...)
	if err := os.WriteFile(filepath.Join(dir, "model.safetensors"), stData, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tokenizer.json"), []byte(`{"model":{"vocab":{"a":0}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for missing embeddings tensor")
	}
}
