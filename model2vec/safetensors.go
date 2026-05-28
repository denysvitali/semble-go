package model2vec

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
)

type tensorInfo struct {
	DType   string   `json:"dtype"`
	Shape   []int    `json:"shape"`
	Offsets [2]int64 `json:"data_offsets"`
}

// Safetensors holds parsed tensor data from a .safetensors file.
type Safetensors struct {
	Tensors map[string][]float32
	Shapes  map[string][]int
}

// LoadSafetensors reads a .safetensors file and returns all tensors as float32.
func LoadSafetensors(path string) (*Safetensors, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 8 {
		return nil, fmt.Errorf("safetensors: file too small")
	}
	hdrLen := binary.LittleEndian.Uint64(data[:8])
	if 8+hdrLen > uint64(len(data)) {
		return nil, fmt.Errorf("safetensors: header extends past file end")
	}
	var meta map[string]json.RawMessage
	if err := json.Unmarshal(data[8:8+hdrLen], &meta); err != nil {
		return nil, fmt.Errorf("safetensors: parse header: %w", err)
	}
	tensors := make(map[string][]float32, len(meta))
	shapes := make(map[string][]int, len(meta))
	buf := data[8+hdrLen:]
	for name, raw := range meta {
		if name == "__metadata__" {
			continue
		}
		var ti tensorInfo
		if err := json.Unmarshal(raw, &ti); err != nil {
			return nil, fmt.Errorf("safetensors: parse %q: %w", name, err)
		}
		start, end := ti.Offsets[0], ti.Offsets[1]
		if end > int64(len(buf)) {
			return nil, fmt.Errorf("safetensors: tensor %q offsets out of range", name)
		}
		slice := buf[start:end]
		f32s, err := decodeDType(slice, ti.DType, ti.Shape)
		if err != nil {
			return nil, fmt.Errorf("safetensors: tensor %q: %w", name, err)
		}
		tensors[name] = f32s
		shapes[name] = ti.Shape
	}
	return &Safetensors{Tensors: tensors, Shapes: shapes}, nil
}

func decodeDType(raw []byte, dtype string, shape []int) ([]float32, error) {
	n := 1
	for _, s := range shape {
		n *= s
	}
	switch dtype {
	case "F32":
		if len(raw) != n*4 {
			return nil, fmt.Errorf("F32 size mismatch: got %d bytes, want %d", len(raw), n*4)
		}
		out := make([]float32, n)
		for i := range out {
			out[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
		}
		return out, nil
	case "F16":
		if len(raw) != n*2 {
			return nil, fmt.Errorf("F16 size mismatch: got %d bytes, want %d", len(raw), n*2)
		}
		out := make([]float32, n)
		for i := range out {
			out[i] = float16ToFloat32(binary.LittleEndian.Uint16(raw[i*2:]))
		}
		return out, nil
	case "BF16":
		if len(raw) != n*2 {
			return nil, fmt.Errorf("BF16 size mismatch: got %d bytes, want %d", len(raw), n*2)
		}
		out := make([]float32, n)
		for i := range out {
			out[i] = bf16ToFloat32(binary.LittleEndian.Uint16(raw[i*2:]))
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported dtype %q", dtype)
	}
}

func float16ToFloat32(h uint16) float32 {
	sign := uint32(h>>15) & 1
	exp := uint32(h>>10) & 0x1F
	frac := uint32(h) & 0x3FF
	if exp == 0 {
		if frac == 0 {
			return math.Float32frombits(sign << 31)
		}
		exp = 1
		for frac&0x400 == 0 {
			frac <<= 1
			exp--
		}
		frac &= 0x3FF
		return math.Float32frombits(sign<<31 | (exp+127-15)<<23 | frac<<13)
	}
	if exp == 0x1F {
		return math.Float32frombits(sign<<31 | 0x7F800000 | frac<<13)
	}
	return math.Float32frombits(sign<<31 | (exp+127-15)<<23 | frac<<13)
}

func bf16ToFloat32(b uint16) float32 {
	return math.Float32frombits(uint32(b) << 16)
}
