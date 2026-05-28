package index

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const embedDim = 256

// candidate pool size taken from each retriever before fusion.
const poolSize = 50

// Index holds the chunked corpus plus its lexical and semantic retrievers.
type Index struct {
	Root   string
	Chunks []Chunk
	vecs   [][]float32
	bm25   *BM25
	emb    Embedder
}

// Result is a scored chunk returned from a query.
type Result struct {
	Chunk Chunk
	Score float64
}

// AllKinds enables code, docs and config indexing.
var AllKinds = map[string]bool{"code": true, "docs": true, "config": true}

// KindsFromContent maps the --content flag value to a kind set.
func KindsFromContent(content string) map[string]bool {
	switch content {
	case "all":
		return AllKinds
	case "docs":
		return map[string]bool{"docs": true}
	case "config":
		return map[string]bool{"config": true}
	default: // code
		return map[string]bool{"code": true}
	}
}

// Open returns an index for root, loading a fresh on-disk cache when the file
// set is unchanged, otherwise (re)building and caching it.
func Open(root string, kinds map[string]bool) (*Index, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	il := loadIgnore(abs)
	files, err := walk(abs, kinds, il)
	if err != nil {
		return nil, err
	}
	fp := fingerprint(files, kinds)
	if idx, ok := loadCache(abs, fp); ok {
		return idx, nil
	}
	idx := build(abs, files)
	_ = saveCache(abs, fp, idx)
	return idx, nil
}

func build(abs string, files []fileInfo) *Index {
	idx := &Index{Root: abs, emb: NewHashEmbedder(embedDim)}
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(abs, f.rel))
		if err != nil || isBinary(data) {
			continue
		}
		idx.Chunks = append(idx.Chunks, chunkFile(f.rel, data)...)
	}
	idx.finalize()
	return idx
}

// finalize builds the retrievers from the chunk list.
func (i *Index) finalize() {
	if i.emb == nil {
		i.emb = NewHashEmbedder(embedDim)
	}
	docTokens := make([][]string, len(i.Chunks))
	for d, c := range i.Chunks {
		docTokens[d] = Tokenize(c.Text + " " + c.File)
	}
	i.bm25 = newBM25(docTokens)
	if i.vecs == nil {
		i.vecs = make([][]float32, len(i.Chunks))
		for d := range i.Chunks {
			i.vecs[d] = i.emb.Embed(docTokens[d])
		}
	}
}

// Search fuses BM25 and semantic rankings via RRF and returns the top-k chunks.
func (i *Index) Search(query string, topK int) []Result {
	if topK <= 0 {
		topK = 10
	}
	qTokens := Tokenize(query)
	lexRanking := topDocs(i.bm25.Score(qTokens))

	qVec := i.emb.Embed(qTokens)
	semScores := map[int]float64{}
	for d, v := range i.vecs {
		if s := cosine(qVec, v); s > 0 {
			semScores[d] = s
		}
	}
	semRanking := topDocs(semScores)

	return i.collect(fuse(lexRanking, semRanking), topK, -1)
}

// FindRelated finds chunks semantically similar to the chunk at file:line,
// excluding chunks from the same file.
func (i *Index) FindRelated(file string, line, topK int) ([]Result, error) {
	if topK <= 0 {
		topK = 10
	}
	src := i.chunkAt(file, line)
	if src < 0 {
		return nil, fmt.Errorf("no indexed chunk at %s:%d", file, line)
	}
	srcFile := i.Chunks[src].File
	qVec := i.vecs[src]
	qTokens := Tokenize(i.Chunks[src].Text)

	semScores := map[int]float64{}
	for d, v := range i.vecs {
		if i.Chunks[d].File == srcFile {
			continue
		}
		if s := cosine(qVec, v); s > 0 {
			semScores[d] = s
		}
	}
	lexScores := map[int]float64{}
	for d, s := range i.bm25.Score(qTokens) {
		if i.Chunks[d].File != srcFile {
			lexScores[d] = s
		}
	}
	ranking := fuse(topDocs(semScores), topDocs(lexScores))
	return i.collect(ranking, topK, src), nil
}

func (i *Index) collect(ranking []scored, topK, exclude int) []Result {
	out := make([]Result, 0, topK)
	for _, s := range ranking {
		if s.doc == exclude {
			continue
		}
		out = append(out, Result{Chunk: i.Chunks[s.doc], Score: s.score})
		if len(out) >= topK {
			break
		}
	}
	return out
}

func (i *Index) chunkAt(file string, line int) int {
	file = filepath.ToSlash(file)
	best := -1
	for d, c := range i.Chunks {
		cf := filepath.ToSlash(c.File)
		if cf != file && !strings.HasSuffix(cf, "/"+file) && cf != strings.TrimPrefix(file, "./") {
			continue
		}
		if line >= c.Start && line <= c.End {
			return d
		}
		if best < 0 {
			best = d
		}
	}
	return best
}

// Savings estimates token cost of reading the whole corpus versus a single
// top-k search result, using a ~4 chars/token heuristic.
type Savings struct {
	Files        int
	Chunks       int
	CorpusTokens int
	SearchTokens int
	Ratio        float64
}

func (i *Index) Estimate(topK int) Savings {
	if topK <= 0 {
		topK = 10
	}
	files := map[string]bool{}
	corpus := 0
	per := make([]int, len(i.Chunks))
	for d, c := range i.Chunks {
		files[c.File] = true
		t := estTokens(c.Text)
		per[d] = t
		corpus += t
	}
	sort.Sort(sort.Reverse(sort.IntSlice(per)))
	search := 0
	for d := 0; d < topK && d < len(per); d++ {
		search += per[d]
	}
	ratio := 0.0
	if search > 0 {
		ratio = float64(corpus) / float64(search)
	}
	return Savings{
		Files:        len(files),
		Chunks:       len(i.Chunks),
		CorpusTokens: corpus,
		SearchTokens: search,
		Ratio:        ratio,
	}
}

func estTokens(s string) int { return (len(s) + 3) / 4 }

func isBinary(data []byte) bool {
	n := len(data)
	if n > 8000 {
		n = 8000
	}
	for _, b := range data[:n] {
		if b == 0 {
			return true
		}
	}
	return false
}

// --- caching ---

type cacheData struct {
	Root        string
	Fingerprint string
	Chunks      []Chunk
	Vecs        [][]float32
}

func cacheDir() string {
	if d := os.Getenv("SEMBLE_CACHE_DIR"); d != "" {
		return d
	}
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "semble-go")
}

func cachePath(abs string) string {
	sum := sha256.Sum256([]byte(abs))
	return filepath.Join(cacheDir(), hex.EncodeToString(sum[:])+".gob")
}

func fingerprint(files []fileInfo, kinds map[string]bool) string {
	sort.Slice(files, func(a, b int) bool { return files[a].rel < files[b].rel })
	h := sha256.New()
	ks := make([]string, 0, len(kinds))
	for k := range kinds {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	h.Write([]byte(strings.Join(ks, ",") + "\n"))
	for _, f := range files {
		_, _ = fmt.Fprintf(h, "%s:%d:%d\n", f.rel, f.mtime, f.size)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func loadCache(abs, fp string) (*Index, bool) {
	f, err := os.Open(cachePath(abs))
	if err != nil {
		return nil, false
	}
	defer func() { _ = f.Close() }()
	var cd cacheData
	if err := gob.NewDecoder(f).Decode(&cd); err != nil {
		return nil, false
	}
	if cd.Fingerprint != fp || len(cd.Vecs) != len(cd.Chunks) {
		return nil, false
	}
	idx := &Index{Root: abs, Chunks: cd.Chunks, vecs: cd.Vecs, emb: NewHashEmbedder(embedDim)}
	idx.finalize()
	return idx, true
}

func saveCache(abs, fp string, idx *Index) error {
	dir := cacheDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := cachePath(abs) + ".tmp" + strconv.Itoa(os.Getpid())
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	cd := cacheData{Root: abs, Fingerprint: fp, Chunks: idx.Chunks, Vecs: idx.vecs}
	if err := gob.NewEncoder(f).Encode(&cd); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, cachePath(abs))
}
