# semble-go

A small, dependency-light Go port of [Semble](https://github.com/MinishLab/semble):
**token-efficient semantic code search** for agents. It indexes a repository into
code-aware chunks and answers queries by returning only the most relevant
snippets — not whole files — so an agent burns a fraction of the tokens it would
spend on `grep` + reading files.

## How it works

```mermaid
flowchart LR
    F[files] --> C[chunk]
    C --> B[BM25 lexical]
    C --> E[embeddings cosine]
    B --> R[Reciprocal Rank Fusion]
    E --> R
    R --> T[top-k chunks]
```

- **Chunking** (`index/chunk.go`): pure-Go, no CGO. Files are split on blank-line
  boundaries with a size cap, keeping chunks roughly aligned to logical blocks.
- **Lexical** (`index/bm25.go`): Okapi BM25 over an identifier-aware tokenizer
  that splits `camelCase`, `snake_case` and digit boundaries.
- **Semantic** (`index/embed.go`): a pure-Go `HashEmbedder` (hashing trick +
  char trigrams) produces dense, L2-normalized vectors compared by cosine.
- **Fusion** (`index/rrf.go`): the two rankings are merged with RRF.
- **Cache** (`index/index.go`): the chunked corpus + vectors are cached under
  `$XDG_CACHE_HOME/semble-go` and reused while the file set is unchanged
  (fingerprinted by path/size/mtime).

The embedder sits behind an `Embedder` interface, so a real Model2Vec backend
(e.g. `potion-code`) can be dropped in later without touching the index — the
default stays pure-Go and offline.

## Install

```sh
go install github.com/denysvitali/semble-go@latest   # installs `semble-go`
# or, from a clone:
make build                                            # builds ./semble
```

## CLI

```sh
semble search "where are http requests retried" ./path  [--top-k 10] [--content code|docs|config|all] [--model /path/to/model2vec]
semble find-related path/to/file.go 42 ./path           [--top-k 10] [--model /path/to/model2vec]
semble savings ./path                                   [--content all] [--model /path/to/model2vec]
semble init [--agent claude|cursor|codex]               # prints MCP config
semble serve                                            # run as MCP server (stdio)
```

`path` defaults to the current directory and may be a local path or a git URL
(remote repos are shallow-cloned and cached).

## MCP server

`semble serve` exposes two read-only tools over stdio:

- **search** — `query`, `repo` (local path or git URL), `top_k`, `content`, `model`
- **find_related** — `file`, `line`, `repo`, `top_k`, `model`

Wire it into an agent with `semble init`:

```jsonc
// .mcp.json
{
  "mcpServers": {
    "semble": { "command": "semble", "args": ["serve"] }
  }
}
```

## Model

By default, semble-go uses a pure-Go `HashEmbedder` (hashing trick over
character trigrams). It works offline with no model files, but it is not
semantically aware — morphologically unrelated synonyms get unrelated vectors.

For real semantic recall, pass `--model /path/to/model2vec` pointing to a local
Model2Vec model directory containing:

- `model.safetensors` — static embedding weights (`[vocab_size, dim]` tensor
  named `embeddings`), supporting F32, F16 and BF16 dtypes.
- `tokenizer.json` — WordPiece tokenizer config (BERT-style, with
  `vocab`, `unk_token`, `continuing_subword_prefix`).

The [MinishLab/potion-models](https://github.com/MinishLab/potion-models)
models (`potion-16M`, `potion-32M`, etc.) are a good fit. Download a model
locally and point `--model` at the directory:

```sh
# example with huggingface-cli
huggingface-cli download minishlab/potion-16M --local-dir ./potion-16M
semble search "http retry logic" --model ./potion-16M
```

The Model2Vec embedder is ~256-dimensional, runs entirely in Go (no CGO), and
the index cache is keyed by embedder ID, so switching between the hash default
and a real model does not invalidate the cache.

## Differences from upstream Semble

- Heuristic chunking instead of tree-sitter (no CGO / per-language grammars).
- Hashing embedder by default; a real Model2Vec backend is available via
  `--model` (see below) for true distilled semantic recall.

## License

MIT — see [LICENSE](LICENSE). This is an independent Go port inspired by
[MinishLab/semble](https://github.com/MinishLab/semble) (MIT, © 2026 Thomas van Dongen).
