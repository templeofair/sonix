// Package ollama — Ollama HTTP client and prompts.
//
// Page vision uses one profile for all configured models: a short German
// imperative with Markdown output, aligned with
// https://ollama.com/Keyvan/german-ocr-turbo ("Extrahiere den Text im Markdown-Format"
// under Ausgabeformate). Translation, summary, and date use ExtractMetadata on
// full joined text.
//
// Requests go to /api/chat by default, including vision. Models published with
// RENDERER/PARSER rather than TEMPLATE are templated by Ollama's renderer,
// which drives the chat path; on /api/generate the behaviour differs
// (ollama#14793) and an untemplated instruct model continues the document
// instead of transcribing it.
package ollama

import (
	"regexp"
	"strings"
)

// pagePromptProfile drives the per-page vision call. Each profile owns its
// name (for logs), its prompt text, and a parser that turns the model's raw
// response into verbatim page text. The endpoint is not part of the profile:
// it is chosen by visionEndpoint() so the bench can A/B chat against generate
// without a second profile.
type pagePromptProfile struct {
	name   string
	prompt string
	// parse takes the raw response text the endpoint produced and returns
	// the verbatim page text.
	parse func(raw string) string
}

// UnifiedVisionProfileName is the stable profile name for per-page vision
// extraction (logged and persisted as engine_id variant).
const UnifiedVisionProfileName = "unified-vision-v1"

// PageProfileNameGermanOCRV1 is kept as an alias for tests and older logs.
// Deprecated: use UnifiedVisionProfileName.
const PageProfileNameGermanOCRV1 = UnifiedVisionProfileName

// VisionPageExtractPrompt is the single user prompt for vision OCR (Markdown
// output). Taken verbatim from https://ollama.com/Keyvan/german-ocr-turbo
// ("Extrahiere den Text im Markdown-Format" under Ausgabeformate).
//
// It is deliberately short because turbo's own Modelfile SYSTEM prompt already
// carries the transcription rules (visible text only, Markdown tables, keep
// number formats, mark illegible spots [unleserlich], no commentary). We send
// no system message on the vision call so that prompt stays in effect.
const VisionPageExtractPrompt = "Extrahiere den Text im Markdown-Format."

// visionPageProfile returns the only page-vision profile (same for all models).
func visionPageProfile() pagePromptProfile {
	return pagePromptProfile{
		name:   UnifiedVisionProfileName,
		prompt: VisionPageExtractPrompt,
		parse:  parseGermanOCRResponse,
	}
}

// parseGermanOCRResponse trims whitespace and strips a single surrounding
// ``` fence if the model wrapped its answer in one. The model is trained
// to emit Markdown plain text, but sampling sometimes adds a fence.
func parseGermanOCRResponse(raw string) string {
	return stripMarkdownFences(strings.TrimSpace(raw))
}

// metadataSystemPrompt is kept only for salvage/legacy comments. Live calls use
// translatePlainSystemPrompt and structuredMetaSystemPrompt below.
//
// Versioning: stored on extractions.prompt_version.
const metadataPromptVersion = "metadata-v12"

// TranslateOnlyPromptVersion is logged when TranslateFullTextEnglish runs; not stored on extractions.
const TranslateOnlyPromptVersion = "translate-only-v8"

// minPageCharsForTranslate skips near-blank pages (e.g. blank reverse sides).
const minPageCharsForTranslate = 40

// mapReduceSummaryChars triggers per-page then final summary when the English
// text is long. Short documents stay on one summary call.
const mapReduceSummaryChars = 6000

// translatePlainSystemPrompt asks for an English Markdown translation only —
// no JSON envelope. Keep this prompt short: long rule lists push some
// OCR/text specialist models into rewrite/echo instead of translation.
// User message is the letter body alone (no "ORIGINAL DOCUMENT TEXT" label).
// Sonix targets German inbound mail; naming German→English is product-generic
// (all German letters), not document-specific.
const translatePlainSystemPrompt = `You are a professional translator. Translate the following German letter into English.

Output ONLY the English translation as Markdown. Every sentence must be English. Do not copy German prose. Keep names, addresses, IDs, and dates as printed.`

// translateRetrySystemPrompt is used when the first pass still looks non-English.
const translateRetrySystemPrompt = `You are a professional translator. The previous attempt was still German. Translate the following German letter into English.

Output ONLY the English translation as Markdown. Every sentence must be English. Do not copy German prose. Keep names, addresses, IDs, and dates as printed.`

// summarizeSystemPrompt asks for a short English summary of the whole document.
const summarizeSystemPrompt = `Summarize the document in 2–4 plain English sentences.

Reply with the summary only — no Markdown, no JSON, no preface. Cover the whole document's purpose and main facts. Never write German.`

// summarizeMapSystemPrompt is used for one page in a map-reduce summary.
const summarizeMapSystemPrompt = `Summarize this page in 1–2 plain English sentences. Reply with the summary only — no Markdown, no JSON. Never write German.`

// documentDateSystemPrompt extracts letterhead date from the original
// (often German) first page. Library category was removed (manual tags only).
const documentDateSystemPrompt = `Extract metadata from the document's letterhead/header.

Return ONE JSON object only (no prose, no fences) with keys:
"document_date": ISO YYYY-MM-DD of the letter's own date from the Datum:/Date: line in the letterhead (zero-pad month and day). If Datum:/Date: is present, always use that value — body deadlines (Freischaltung, Gültig bis, Zahlungsziel, etc.) are never the document date. Use "" only when no letterhead Datum:/Date: exists. Never invent.

Use "" for missing fields; never null.`

const documentDateFormat = `{
  "type": "object",
  "properties": {
    "document_date": { "type": "string" }
  },
  "required": ["document_date"]
}`

// structuredMetaSystemPrompt is kept for degrade/legacy single-field salvage paths.
const structuredMetaSystemPrompt = `Extract metadata from the English document text.

Return ONE JSON object only (no prose, no fences) with keys:
"summary": 2–4 plain English sentences on purpose and main facts (no Markdown). Never write the summary in German.
"document_date": ISO YYYY-MM-DD of the letter's own date from the letterhead/header (near Date:/Datum:), or "" if absent/ambiguous. Ignore deadlines and validity dates in the body. Never invent.

Use "" for missing fields; never null.`

const metadataMultiPageSupplement = `Multi-page: prefer the first page's letterhead for document_date; write the summary from the full document.`

const structuredMetaFormat = `{
  "type": "object",
  "properties": {
    "summary": { "type": "string" },
    "document_date": { "type": "string" }
  },
  "required": ["summary", "document_date"]
}`

// coerceISODate normalizes a model-produced date string to strict
// YYYY-MM-DD or empty. The metadata system prompt asks for ISO, but
// models often emit German DD.MM.YYYY or loosely padded ISO (2026-7-10).
// Rather than relax the downstream contract — the UI and DB index expect
// zero-padded ISO — we coerce in code so that looser behaviour doesn't leak.
//
// Recognized inputs:
//   - ISO (optional zero-pad): 2024-12-19 or 2026-7-10
//   - German DD.MM.YYYY:     19.12.2024
//   - German short DD.MM.YY: 19.12.24  (assumed 20YY; pre-2000 dates
//     are not a realistic target for
//     this app's letter scanning)
//   - DD/MM/YYYY:            19/12/2024
//
// Anything else returns "" so the field stays refusal-safe rather than
// shipping a garbled date.
func coerceISODate(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if m := looseISODateRe.FindStringSubmatch(s); len(m) == 4 {
		return formatISO(m[1], m[2], m[3])
	}
	if m := germanLongDateRe.FindStringSubmatch(s); len(m) == 4 {
		return formatISO(m[3], m[2], m[1])
	}
	if m := germanShortDateRe.FindStringSubmatch(s); len(m) == 4 {
		return formatISO("20"+m[3], m[2], m[1])
	}
	if m := slashDateRe.FindStringSubmatch(s); len(m) == 4 {
		return formatISO(m[3], m[2], m[1])
	}
	return ""
}

// formatISO returns YYYY-MM-DD with zero-padded month and day, or ""
// if the components are not plausible (sanity check: month 1-12, day
// 1-31). We don't validate calendar correctness (Feb 30 etc.) — that's
// false-precision for a free-text input from a vision model.
func formatISO(y, m, d string) string {
	if len(m) == 1 {
		m = "0" + m
	}
	if len(d) == 1 {
		d = "0" + d
	}
	mi, di := atoiOrZero(m), atoiOrZero(d)
	if mi < 1 || mi > 12 || di < 1 || di > 31 {
		return ""
	}
	return y + "-" + m + "-" + d
}

func atoiOrZero(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

var (
	// looseISODateRe accepts zero-padded or unpadded month/day (2026-7-10).
	looseISODateRe    = regexp.MustCompile(`^(\d{4})-(\d{1,2})-(\d{1,2})$`)
	germanLongDateRe  = regexp.MustCompile(`^(\d{1,2})\.(\d{1,2})\.(\d{4})$`)
	germanShortDateRe = regexp.MustCompile(`^(\d{1,2})\.(\d{1,2})\.(\d{2})$`)
	slashDateRe       = regexp.MustCompile(`^(\d{1,2})/(\d{1,2})/(\d{4})$`)
	// letterheadDatumRe / letterheadDateRe find the letter's own date label.
	letterheadDatumRe = regexp.MustCompile(`(?im)^\s*Datum\s*:\s*(\d{1,2}\.\d{1,2}\.\d{2,4})\b`)
	letterheadDateRe  = regexp.MustCompile(`(?im)^\s*Date\s*:\s*(\d{1,2}[./]\d{1,2}[./]\d{2,4})\b`)
)

// letterheadDocumentDate pulls Datum:/Date: from the first-page letterhead when
// the model returns empty or an unparseable document_date. Prefers German Datum.
func letterheadDocumentDate(page string) string {
	if m := letterheadDatumRe.FindStringSubmatch(page); len(m) == 2 {
		return coerceISODate(m[1])
	}
	if m := letterheadDateRe.FindStringSubmatch(page); len(m) == 2 {
		return coerceISODate(m[1])
	}
	return ""
}

// salvageMetadata is a tolerant fallback for ExtractMetadata responses
// that fail strict JSON parsing. Specialist OCR models sometimes hallucinate
// repeated empty rows that exhaust the model's context budget and truncate
// the response mid-string, leaving a half-formed JSON value. Rather than
// fail the whole extraction in that case, we walk the raw content with a
// permissive scanner and return whatever fields we can recover. Returns
// empty strings for fields that are absent or unparseable; the caller is
// expected to fall back to original text for english when it is empty.
//
// This is best-effort. If nothing is salvageable, all returns are empty
// and the caller should still treat that as a failure.
func salvageMetadata(raw string) (summary, english, date string) {
	// document_date is short and unambiguous: ISO YYYY-MM-DD only.
	if m := dateFieldRe.FindStringSubmatch(raw); len(m) == 2 {
		date = m[1]
	}
	summary = salvageStringField(raw, "summary")
	english = salvageStringField(raw, "full_text_english")
	return
}

// dateFieldRe matches "document_date": "YYYY-MM-DD" with tolerant
// whitespace. The strict 4-2-2 digit shape avoids accidentally matching
// dates embedded in the document text rather than in the date field.
var dateFieldRe = regexp.MustCompile(`"document_date"\s*:\s*"(\d{4}-\d{2}-\d{2})"`)

// salvageStringField extracts the value of a JSON string field with a
// tolerant scanner. Handles \" inside the value, accepts truncated values
// (returns whatever was read so far), and decodes the common JSON escape
// sequences (\n, \t, \r, \", \\, \/) that the strict decoder would have
// applied. Unknown escapes are dropped — the data is corrupt, the goal is
// to extract a usable approximation, not to round-trip exactly.
func salvageStringField(raw, key string) string {
	needle := `"` + key + `"`
	idx := strings.Index(raw, needle)
	if idx < 0 {
		return ""
	}
	rest := raw[idx+len(needle):]
	rest = strings.TrimLeft(rest, " \t\n\r")
	if !strings.HasPrefix(rest, ":") {
		return ""
	}
	rest = strings.TrimLeft(rest[1:], " \t\n\r")
	if !strings.HasPrefix(rest, `"`) {
		return ""
	}
	rest = rest[1:]
	var b strings.Builder
	escape := false
	for i := 0; i < len(rest); i++ {
		c := rest[i]
		if escape {
			switch c {
			case '"', '\\', '/':
				b.WriteByte(c)
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			}
			escape = false
			continue
		}
		if c == '\\' {
			escape = true
			continue
		}
		if c == '"' {
			return b.String()
		}
		b.WriteByte(c)
	}
	return strings.TrimSpace(b.String())
}

// stripMarkdownFences removes a single surrounding ``` block if present.
// Leaves interior fences untouched; callers that need recursive stripping
// should call multiple times.
func stripMarkdownFences(s string) string {
	if !strings.HasPrefix(s, "```") {
		return s
	}
	// Drop the opening fence line (```, ```markdown, ```text, …).
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[i+1:]
	} else {
		return strings.TrimPrefix(s, "```")
	}
	// Drop the final fence if present.
	if j := strings.LastIndex(s, "```"); j >= 0 {
		s = s[:j]
	}
	return strings.TrimSpace(s)
}
