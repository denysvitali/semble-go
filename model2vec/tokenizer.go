package model2vec

import (
	"encoding/json"
	"os"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// WordPiece tokenizer compatible with BERT-style models.
type WordPiece struct {
	Vocab                map[string]int32
	InvVocab             map[int32]string
	UnkToken             string
	ContinuingSubword    string // "##" for BERT
	MaxInputCharsPerWord int
}

// LoadTokenizer reads a tokenizer.json file and returns a WordPiece tokenizer.
func LoadTokenizer(path string) (*WordPiece, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg struct {
		Model struct {
			Vocab                map[string]int32 `json:"vocab"`
			UnkToken             string           `json:"unk_token"`
			ContinuingSubword    string           `json:"continuing_subword_prefix"`
			MaxInputCharsPerWord int              `json:"max_input_chars_per_word"`
		} `json:"model"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	wp := &WordPiece{
		Vocab:                cfg.Model.Vocab,
		InvVocab:             make(map[int32]string, len(cfg.Model.Vocab)),
		UnkToken:             cfg.Model.UnkToken,
		ContinuingSubword:    cfg.Model.ContinuingSubword,
		MaxInputCharsPerWord: cfg.Model.MaxInputCharsPerWord,
	}
	if wp.UnkToken == "" {
		wp.UnkToken = "[UNK]"
	}
	if wp.ContinuingSubword == "" {
		wp.ContinuingSubword = "##"
	}
	if wp.MaxInputCharsPerWord <= 0 {
		wp.MaxInputCharsPerWord = 100
	}
	for tok, id := range wp.Vocab {
		wp.InvVocab[id] = tok
	}
	return wp, nil
}

// Tokenize normalizes and tokenizes text into subword token IDs.
func (wp *WordPiece) Tokenize(text string) []int32 {
	words := preTokenize(text)
	var ids []int32
	for _, word := range words {
		ids = append(ids, wp.tokenizeWord(word)...)
	}
	return ids
}

func (wp *WordPiece) tokenizeWord(word string) []int32 {
	if len([]rune(word)) > wp.MaxInputCharsPerWord {
		if id, ok := wp.Vocab[wp.UnkToken]; ok {
			return []int32{id}
		}
		return nil
	}
	var ids []int32
	start := 0
	runes := []rune(word)
	for start < len(runes) {
		end := len(runes)
		var found string
		for end > start {
			sub := string(runes[start:end])
			if start > 0 {
				sub = wp.ContinuingSubword + sub
			}
			if _, ok := wp.Vocab[sub]; ok {
				found = sub
				break
			}
			end--
		}
		if found == "" {
			if id, ok := wp.Vocab[wp.UnkToken]; ok {
				ids = append(ids, id)
			}
			return ids
		}
		ids = append(ids, wp.Vocab[found])
		start = end
	}
	return ids
}

// preTokenize splits on whitespace and punctuation (BERT style).
func preTokenize(text string) []string {
	text = strings.ToLower(text)
	text = stripAccents(text)
	var words []string
	var current []rune
	for _, r := range text {
		switch {
		case unicode.IsSpace(r):
			if len(current) > 0 {
				words = append(words, string(current))
				current = nil
			}
		case isPunct(r):
			if len(current) > 0 {
				words = append(words, string(current))
				current = nil
			}
			words = append(words, string(r))
		default:
			current = append(current, r)
		}
	}
	if len(current) > 0 {
		words = append(words, string(current))
	}
	return words
}

func isPunct(r rune) bool {
	if r > 0x7F {
		return false
	}
	return unicode.IsPunct(r)
}

func stripAccents(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(s) {
		if r == 'ß' {
			b.WriteString("ss")
			continue
		}
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
