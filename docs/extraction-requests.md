# Extraction — LLM requests walkthrough

**Who this is for:** Developers debugging Ollama traffic or comparing branches side by side.

**In one sentence:** For a **single page image X** (and **document X₂** with two pages when multi-page behavior matters), this doc shows which Go helper runs, which HTTP endpoint Ollama receives, and a **representative JSON body** (pretty-printed).

**Conventions:** Replace `"<base64-of-image-X>"` with real base64 from a page file. Prompt strings match [`internal/ollama/prompts.go`](../internal/ollama/prompts.go) and request assembly in [`internal/ollama/client.go`](../internal/ollama/client.go). **`prompts.go` is the source of truth** when you suspect doc drift.

---

## 1. `use_ocr: true` (single page)

**What happens:** Tesseract reads the image (no LLM). Then **`ExtractMetadata`** runs on the combined original text.

| Step | Function | Endpoint |
|------|----------|----------|
| Page text | `ocrTextExtractor.extract` | *(none — local OCR)* |
| Summary + English + date | `Client.ExtractMetadata` | `POST /api/chat` |

**Example — translate** (`TextModel` e.g. `Keyvan/german-text-3.1`), plain text, **no** `format`:

```json
{
  "model": "Keyvan/german-text-3.1",
  "stream": false,
  "think": false,
  "options": {
    "temperature": 0,
    "top_k": 1,
    "repeat_penalty": 1.1,
    "num_ctx": 8192,
    "num_predict": 1024
  },
  "messages": [
    {
      "role": "system",
      "content": "<translatePlainSystemPrompt from prompts.go — translate-only-v6>"
    },
    {
      "role": "user",
      "content": "ORIGINAL DOCUMENT TEXT:\n<text from Tesseract>"
    }
  ]
}
```

**Example — structured summary/date** (JSON Schema `format`):

```json
{
  "model": "Keyvan/german-text-3.1",
  "stream": false,
  "think": false,
  "format": {
    "type": "object",
    "properties": {
      "summary": { "type": "string" },
      "document_date": { "type": "string" }
    },
    "required": ["summary", "document_date"]
  },
  "options": {
    "temperature": 0,
    "top_k": 1,
    "repeat_penalty": 1.1,
    "num_ctx": 8192,
    "num_predict": 1024
  },
  "messages": [
    {
      "role": "system",
      "content": "<structuredMetaSystemPrompt from prompts.go — metadata-v9>"
    },
    {
      "role": "user",
      "content": "DOCUMENT TEXT:\n<original text>"
    }
  ]
}
```

`num_ctx` and `num_predict` are **sized from the input length**, so the values above vary per document; see `textOptions` in [`options.go`](../internal/ollama/options.go).

If the structured call fails, translation is still kept; summary/date degrade to narrower calls then empty fields.

---

## 2. Vision two-phase (`use_ocr: false`)

**What happens:** For each page, **`ExtractPage`** → **`callVision`** → **`POST /api/chat`** with the image on the user message and `VisionPageExtractPrompt`. Then **`ExtractMetadata`** on the full joined text. For **X₂**, the user message uses `SECTION "FIRST_PAGE"` / `SECTION "ALL_PAGES"` (see `ExtractMetadata` in `client.go`).

**Per-page vision:**

```json
{
  "model": "Keyvan/german-ocr-turbo",
  "stream": false,
  "think": false,
  "options": {
    "temperature": 0,
    "top_k": 1,
    "repeat_penalty": 1.1,
    "num_ctx": 8192,
    "num_predict": 4096
  },
  "messages": [
    {
      "role": "user",
      "content": "Extrahiere den Text im Markdown-Format.",
      "images": ["<base64-of-image-X>"]
    }
  ]
}
```

**No `system` message on the vision call.** Turbo's own Modelfile `SYSTEM` prompt carries the transcription rules (visible text only, Markdown tables, keep number formats, mark illegible spots `[unleserlich]`, no commentary). A request-level system message would replace it, which is how the 3.x "Prompt-Edition" models lose the only thing they add.

**Metadata** — same structure as branch 1; for two pages, `system` includes `metadataMultiPageSupplement` and `user` lists both sections.

---

## 3. Why vision uses `/api/chat`

Vision used to post `/api/generate`. It now uses `/api/chat`, and the `think` field is always **top-level**.

| Concern | Why it matters |
|---------|----------------|
| **Endpoint** | Models published with `RENDERER`/`PARSER` instead of a `TEMPLATE` (turbo declares `RENDERER qwen3-vl-instruct`) are templated by Ollama's renderer, which drives the chat path. The two endpoints are not equivalent for such models ([ollama#14793](https://github.com/ollama/ollama/issues/14793)); an instruct model that never receives its chat wrapping **continues the document** rather than transcribing it, producing a page repeated several times over. |
| **`think` placement** | Ollama silently ignores unrecognized keys inside `options`, so `options:{think:false}` is a no-op that looks correct. On a thinking model the whole `num_predict` budget then goes to reasoning tokens that Ollama strips out of `message.content`, leaving it empty — reported downstream as `unexpected end of JSON input`. See [ollama#14716](https://github.com/ollama/ollama/issues/14716) and [ollama#13353](https://github.com/ollama/ollama/issues/13353). |
| **Explicit budgets** | Omitting `num_ctx` / `num_predict` inherits the model's Modelfile values. Turbo ships `num_ctx 4096` / `num_predict 2048`, so a multi-page document overflowed the window and a dense page truncated mid-transcription — silently, because `done_reason` was discarded. |
| **Greedy decode** | Turbo ships `temperature 0.1, top_k 20, top_p 0.9`, so the same page can transcribe differently twice. Transcription and translation want the most likely token and a reproducible result. |
| **`repeat_penalty`** | Turbo ships `1.5`. The penalty only looks back over `repeat_last_n` tokens (64 by default), so it cannot stop a page-length repeat, while it does push the model away from legitimately repeated wording. Lowered to 1.1; long-range repetition is detected separately. |

To compare endpoints on one image, set `OLLAMA_VISION_ENDPOINT=/api/generate`.

---

## Related

- **[extraction.md](extraction.md)** — Pipeline map and timing logs.
- **[configuration.md](configuration.md)** — Env vars.
