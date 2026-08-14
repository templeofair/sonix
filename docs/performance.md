# Extraction — performance and quality audit

**Who this is for:** Operators sizing hardware, tuning Ollama, or prioritizing code changes without sacrificing letter quality.

**In one sentence:** This document ties Sonix’s **sequential, multi-round-trip** extraction design to **realistic** home-lab hardware, external best practices, and **measurable** quality checks.

---

## Executive summary

- **Context:** Sonix processes **one page at a time** for vision or OCR, then **one** metadata pass. Wall time grows roughly with **pages × (per-page vision or OCR latency) + metadata latency** (plus optional translate-only retry).
- **Reference box:** A **hybrid Intel mini-PC class** machine (P+E cores, integrated graphics, **no discrete GPU**, **~32 GB RAM**) is a realistic home-lab baseline. Treat **CPU + RAM bandwidth** as the bottleneck until measured.
- **Ollama reality:** Default Ollama does **not** route GGUF inference through the **NPU**. It may use **CPU** and/or **iGPU** depending on build and drivers — check **`ollama ps`** *Processor* per [Ollama FAQ](https://docs.ollama.com/faq). Treat **CPU + RAM bandwidth** as the default bottleneck until measured otherwise.
- **Top levers (low code risk):** Use **`OLLAMA_KEEP_ALIVE`** and server-side parallelism settings so models are not cold-loaded every document; align **`OLLAMA_NUM_PARALLEL`** / queue with **`EXTRACTION_MAX_CONCURRENT`** (default 1).
- **How to measure:** Use **`extraction_timing`** logs ([extraction.md](extraction.md)) and a **fixed set of PDFs/images** as a before/after benchmark; re-check **legibility, translation fidelity, and date accuracy** after any change. For latency claims, use **warm-only** runs (below) — never mix cold model load into a baseline.

---

## Warm-only measurement (Demo / baselines)

Cold vision loads can add ~100s+ of `prefill_ms` and are **ops noise** (keep-alive), not product regressions. Exclude them from wall-time baselines.

**Preflight**

- Set Ollama **`OLLAMA_KEEP_ALIVE`** so vision and text models stay loaded; optionally `OLLAMA_MAX_LOADED_MODELS=2` when using two models.
- Confirm both models appear in `ollama ps` before Extract.
- On the first page of the run, vision **`prefill_ms` should be under ~1s**. If it is tens of seconds, discard the run for latency claims and warm up again.

**Golden letter checklist** (a fixed sample document you keep for regression)

| Check | Pass |
|-------|------|
| Status | `ready` with English body, **or** honest `partial` with original kept (never German labelled as English) |
| `full_text_english` | Predominantly English (spot-check; no large `Sehr geehrte` / German body blocks) |
| `document_date` | `2026-07-10` |
| Translate calls | Prefer one `purpose=translate` when first pass succeeds; two only if strong-retry ran |
| Cue | No outer full-doc retry solely because English keeps a German street suffix |
| Wall (informational) | Target ~110–120s warm once single translate; do not sacrifice date/translation quality for speed |

**After any translate or date change:** re-run this checklist on a warm Demo before claiming improvement. Date path (ISO coerce + letterhead `Datum:` fallback) and cue heuristic must stay green — see unit tests under `internal/ollama/`.

### Frozen quality baseline (2026-07-25, post translate-only-v8)

Warm sample (`vision` + text models already loaded):

| Metric | Result |
|--------|--------|
| Status | `ready` |
| English | Real English (`translate-only-v8`, **1** translate call) |
| `document_date` | `2026-07-10` |
| Translate wall | ~31s |
| Summary + date | ~10s + ~8s |
| Vision page wall | ~174s (Ollama `decode_ms` ~62s, `prefill_ms` ~100ms — residual is prep/transfer/queue; **next perf target**) |
| Total wall | ~223s |

**Quality reset goal met** for translation + date. Wall-time target ~110–120s remains a **vision/ops** follow-up (keep both models loaded; measure `vision_timing prep_ms` / `http_ms` in logs; optional `OLLAMA_VISION_MAX_EDGE` 1536 vs 2048 A/B). Do not regress fail-closed translate or letterhead date to chase speed.

---

## 1. Workload shaping (Sonix)

Sonix controls **how many** LLM calls and **how large** each context is:

- **OCR path** (`use_ocr`): zero vision tokens; then translate + summary + document date (plus optional outer echo retry).
- **Vision path:** **N** per-page `/api/chat` calls (image on the user message), then translate + summary + document date.

**Map:** See the LLM load table in [extraction.md](extraction.md). **Timing logs** validate the table on your deployment.

---

## 2. Ollama server tuning (outside Sonix)

Topics that matter for queueing, memory, and concurrency (details change with Ollama versions — verify current docs):

- **`OLLAMA_MAX_LOADED_MODELS`**, **`OLLAMA_NUM_PARALLEL`**, **`OLLAMA_MAX_QUEUE`**, **`OLLAMA_KEEP_ALIVE`** — [Ollama FAQ](https://docs.ollama.com/faq).
- **Context length vs memory** — larger contexts reduce concurrent loads on the same machine.
- **Thread / CPU behavior** — throughput is often **memory-bandwidth** limited; do not expect linear speedup from hyperthreads alone ([discussion context](https://github.com/ollama/ollama/issues/2496)).

**NPU / iGPU caveat:** On a hybrid Intel mini-PC class host, default Ollama does **not** offload LLM inference to an NPU. An iGPU may participate for some workloads — **measure** with `ollama ps`, do not assume.

---

## 3. Model selection and quantization

- **Vision-capable** model required for the default path; all models use the same **short German Markdown-style** `/api/chat` prompt (see `internal/ollama/prompts.go`). Quality still varies by model—validate on real scans.
- **Smaller / more quantized** GGUF variants run faster on CPU but can **hurt** OCR fidelity or JSON conformance. Treat quantization as an **A/B** decision on real scans: [llama.cpp quantization](https://github.com/ggerganov/llama.cpp) background; recent survey [arXiv:2601.14277](https://arxiv.org/html/2601.14277v1) (*Which Quantization Should I Use?*) discusses throughput vs quality tradeoffs on CPU.

**Quality re-check:** After changing model or tag, verify **verbatim text** (layout, numbers), **English** (no echoed German), and **document_date** on your sample set.

---

## 4. Retries and tail latency

**Per-page retries:** Up to **3** attempts on transient vision/OCR failures before the document fails (`extractPageWithRetry`). Timing logs emit **`page_step`** after each successful page.

**Metadata / translate-only:** A slow or large-context metadata call dominates tail latency on big documents; see **Large documents** in [extraction.md](extraction.md).

---

## 5. Reliability and overload

- **Timeouts:** **`OLLAMA_TIMEOUT_MINUTES`** caps each HTTP call; **`EXTRACTION_JOB_TIMEOUT_MINUTES`** (default 60) caps the whole document job. Too-low per-call timeouts cause false failures on slow hosts.
- **503 / queue:** Under load, Ollama may queue or reject; see FAQ. Sonix surfaces failures as **failed** extraction with a stored error.

**Quality note:** Aggressive timeouts can mark good documents failed — balance against operator patience.

---

## 6. Future engineering (optional)

Labeled **not in current code**: per-request trace IDs to logs, optional **`postJSON`** timing labels in `client.go`, metrics export, **smaller “metadata-only” model** slot — only after baseline **`extraction_timing`** numbers exist.

---

## Sonix-specific recommendation buckets

| Bucket | Examples |
|--------|----------|
| **Near-zero code risk** | Use `use_ocr` for text-only pipelines or A/B vs vision; set Ollama **keep-alive**; single parallel generation if RAM is tight. |
| **Operator process** | Fixed corpus + `grep extraction_timing` before/after; document model tag in run notes. |
| **Code follow-ups** | Optional finer-grained Ollama timing in `client.go` once server-level phases are understood. |

---

## Quality guardrails (mandatory)

Every performance change should note **what to re-check**:

1. **Legibility** — Verbatim text matches the scan (no dropped lines; tables/lists preserved where expected).
2. **Translation** — No large blocks of source language left in `full_text_english`; proper nouns acceptable as documented in prompts.
3. **Summary and date** — Summary matches content; **document_date** is plausible ISO or empty, not hallucinated.

**German business documents:** A smaller or different vision model may **hurt** layout-sensitive OCR — **measure**, do not assume.

---

## Cross-links

- Architecture: [extraction.md](extraction.md)
- Configuration: [configuration.md](configuration.md)
- Request shapes: [extraction-requests.md](extraction-requests.md)

---

## Footnotes

1. Ollama FAQ (processors, environment): [docs.ollama.com/faq](https://docs.ollama.com/faq).
2. Experimental CPU preload PR (may not ship): [ollama/ollama#10596](https://github.com/ollama/ollama/pull/10596) — **investigate only**, not a dependency.
