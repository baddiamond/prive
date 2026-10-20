# 0009 — Hybrid local search (full-text + semantic)

- **Status:** accepted (planned — see [roadmap](../ROADMAP.md#future-features))
- **Date:** 2026-10-20

## Context

Search across every account and network, on this computer only, is a core promise (⌘K, `docs/UI.md` §2.5).
Exact-word search misses what people actually remember ("the thing about the rent" vs. "apartment payment").
Single messages ("ok", "sure") carry little meaning on their own. Search must run on basic computers.

## Decision

Two retrievers over the same encrypted database ([ADR 0006](0006-sqlite-driver-ncruces.md)),
merged with **Reciprocal Rank Fusion (RRF)**:

### 1. Full-text (always on)

- **FTS5** with BM25 ranking and the `unicode61 remove_diacritics 2` tokenizer
  ("cancion" finds "canción").
- Indexes message text, captions, file names and transcripts ([ADR 0008](0008-local-voice-transcription.md)).
- Created from the first store migration, so it never needs a rebuild later.

### 2. Semantic (opt-in)

- Embeddings from **Model2Vec `potion-multilingual-128M`** (MIT; distilled from `bge-m3`; 101 languages;
  256 dimensions). It is a static model: tokenize and average vectors from a table, no neural network at query
  time. We write a small Go loader (tokenizer + table lookup); the weights are memory-mapped, so RAM use stays low.
- **Conversation windows**, not single messages: consecutive messages of one chat, split on long silences
  or a size limit. Each window points back to its messages, and a hit opens the best message in context.
- Vectors stored with **`sqlite-vec`**, quantized to int8 (binary quantization if a dataset needs it).
- Off by default. ⌘K offers it once, when a search returns few results; it is also in Settings.
  It needs one model download under the same rules as ADR 0008 (explicit action, pinned SHA-256, import from file).
- Possible later upgrade (opt-in): `multilingual-e5-small` (MIT) on ONNX Runtime for higher quality.
- To evaluate: pruning the vocabulary to the shipped languages to shrink the download.

### Query features

- Filters: `from:`, `in:`, `account:`, `type:` (e.g. `type:voice`), `before:` / `after:`, across all accounts.
- Result groups in the palette: **Chats**, **Messages**, **Voice notes**. Semantic-only hits carry a
  "Similar meaning" label because there is no exact match to highlight.
- Chats excluded from search, locked or hidden follow [ADR 0010](0010-chat-and-account-privacy.md).

### Indexing

- New messages are indexed as they are stored. The initial backfill runs in the background at low priority,
  can be paused, and resumes after restart.

## Consequences

- Exact search works everywhere from day one at almost no cost (index ≈ 30–50 % of text size).
- Semantic search is cheap in CPU and RAM, so it fits basic computers; its cost is a ~130 MB download.
- The index lives inside the encrypted database, so it is as private as the messages.
- New translation keys under `search.*`.

## Alternatives considered

- **Full-text only** — misses paraphrases and cross-language matches.
- **Transformer embeddings by default** (e5, EmbeddingGemma) — better quality but heavier at index time;
  EmbeddingGemma's license is not on our allow-list ([ADR 0011](0011-licensing-policy.md)).
- **External search engine** (Meilisearch, Tantivy service, Elasticsearch) — another process and dependency.
- **Per-message embeddings** — short messages produce noisy vectors.
