package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Client calls the Ollama API (vision and generate).
type Client struct {
	BaseURL     string
	VisionModel string
	TextModel   string
	HTTPClient  *http.Client
}

// NewClient returns a client with the given base URL (e.g. http://localhost:11434) and model names.
// HTTP timeout is OLLAMA_TIMEOUT_MINUTES env (default 15). Increase for slow Ollama or large models.
func NewClient(baseURL, visionModel, textModel string) *Client {
	if visionModel == "" {
		visionModel = "llava"
	}
	if textModel == "" {
		textModel = "llama3.2"
	}
	timeout := 15 * time.Minute
	if m := os.Getenv("OLLAMA_TIMEOUT_MINUTES"); m != "" {
		if n, err := strconv.Atoi(m); err == nil && n > 0 {
			timeout = time.Duration(n) * time.Minute
		}
	}
	return &Client{
		BaseURL:     baseURL,
		VisionModel: visionModel,
		TextModel:   textModel,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// noThink is the address-able false we send as the top-level "think" field.
//
// This must be top-level, never inside Options. Ollama silently ignores
// unrecognized option keys, so `options:{think:false}` looks correct and does
// nothing (ollama#14793). On a thinking model the consequence is severe rather
// than cosmetic: the whole num_predict budget goes to reasoning tokens which
// Ollama then strips out of the reply, leaving content empty and json.Unmarshal
// reporting "unexpected end of JSON input" — the failure this fixes.
// Non-thinking models ignore the field harmlessly.
var noThink = false

// chatRequest and chatResponse match Ollama /api/chat.
//
// Format and Options are optional. Format="json" asks Ollama to constrain
// the model's output to valid JSON (a transport-layer guarantee, separate
// from anything the prompt says). Options carries generation knobs like
// temperature; we keep this map-typed so callers can add fields without
// touching this struct.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Think    *bool         `json:"think,omitempty"`
	// Format may be the string "json" or a JSON Schema object (json.RawMessage).
	Format  any            `json:"format,omitempty"`
	Options map[string]any `json:"options,omitempty"`
}

type chatMessage struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"`
}

// callStats carries the response metadata Ollama already sends and we used to
// throw away. Without done_reason a truncated reply is indistinguishable from
// a complete one, and without the token counts we cannot tell whether a slow
// call was dominated by reading the image (prefill) or writing the answer
// (decode) — which is the difference between an image-size problem and an
// output-length problem.
type callStats struct {
	Endpoint     string
	Model        string
	DoneReason   string
	PromptTokens int
	EvalTokens   int
	PrefillMs    int64
	DecodeMs     int64
	ContentLen   int
	ThinkingLen  int
}

type chatResponse struct {
	Message struct {
		Content string `json:"content"`
		// Thinking is where a thinking model's output lands. Decoding it
		// means an empty content can be reported as "the model spent its
		// budget reasoning" rather than as an unexplained parse failure.
		Thinking string `json:"thinking"`
	} `json:"message"`
	DoneReason         string `json:"done_reason"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	EvalCount          int    `json:"eval_count"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
	EvalDuration       int64  `json:"eval_duration"`
}

func (r chatResponse) stats(endpoint, model string) callStats {
	return callStats{
		Endpoint:     endpoint,
		Model:        model,
		DoneReason:   r.DoneReason,
		PromptTokens: r.PromptEvalCount,
		EvalTokens:   r.EvalCount,
		PrefillMs:    r.PromptEvalDuration / 1e6,
		DecodeMs:     r.EvalDuration / 1e6,
		ContentLen:   len(r.Message.Content),
		ThinkingLen:  len(r.Message.Thinking),
	}
}

// generateRequest and generateResponse match Ollama /api/generate. Retained so
// the bench can compare endpoints on one image; /api/chat is the default (see
// visionEndpoint).
type generateRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Images  []string       `json:"images"`
	Stream  bool           `json:"stream"`
	Think   *bool          `json:"think,omitempty"`
	Options map[string]any `json:"options,omitempty"`
}

type generateResponse struct {
	Response           string `json:"response"`
	Thinking           string `json:"thinking"`
	DoneReason         string `json:"done_reason"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	EvalCount          int    `json:"eval_count"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
	EvalDuration       int64  `json:"eval_duration"`
}

func (r generateResponse) stats(endpoint, model string) callStats {
	return callStats{
		Endpoint:     endpoint,
		Model:        model,
		DoneReason:   r.DoneReason,
		PromptTokens: r.PromptEvalCount,
		EvalTokens:   r.EvalCount,
		PrefillMs:    r.PromptEvalDuration / 1e6,
		DecodeMs:     r.EvalDuration / 1e6,
		ContentLen:   len(r.Response),
		ThinkingLen:  len(r.Thinking),
	}
}

// logCall emits one structured line per Ollama call. Grep: ollama_call.
func logCall(purpose string, s callStats, extra ...string) {
	tokPerSec := 0.0
	if s.DecodeMs > 0 {
		tokPerSec = float64(s.EvalTokens) / (float64(s.DecodeMs) / 1000)
	}
	line := fmt.Sprintf(
		"ollama_call purpose=%s model=%q endpoint=%s prompt_tokens=%d eval_tokens=%d tok_s=%.1f prefill_ms=%d decode_ms=%d done_reason=%q content_len=%d thinking_len=%d",
		purpose, s.Model, s.Endpoint, s.PromptTokens, s.EvalTokens, tokPerSec,
		s.PrefillMs, s.DecodeMs, s.DoneReason, s.ContentLen, s.ThinkingLen,
	)
	if len(extra) > 0 {
		line += " " + strings.Join(extra, " ")
	}
	log.Print(line)
}

// ErrEmptyContent means the model returned no visible text. Distinguished from
// a parse failure because the cause and the fix are different: an empty reply
// is a thinking model or an exhausted budget, not malformed JSON.
type ErrEmptyContent struct {
	Purpose     string
	DoneReason  string
	ThinkingLen int
}

func (e *ErrEmptyContent) Error() string {
	if e.ThinkingLen > 0 {
		return fmt.Sprintf(
			"%s: model returned empty content after %d chars of reasoning (done_reason=%q); send think:false or use an -instruct model",
			e.Purpose, e.ThinkingLen, e.DoneReason)
	}
	return fmt.Sprintf("%s: model returned empty content (done_reason=%q)", e.Purpose, e.DoneReason)
}

// truncatedError reports a reply cut short by the output budget.
func truncatedError(purpose string, s callStats) error {
	return fmt.Errorf("%s: output truncated at %d tokens (done_reason=%q); raise num_predict",
		purpose, s.EvalTokens, s.DoneReason)
}

func isTruncated(s callStats) bool { return s.DoneReason == "length" }

// ErrUnparseable carries the raw model output that failed to parse so the
// caller can persist it to extractions.raw_response. Debugging a prompt from a
// log line only works if the logs are still around; the column exists for
// exactly this and was never being written.
type ErrUnparseable struct {
	Purpose string
	Raw     string
	Err     error
}

func (e *ErrUnparseable) Error() string {
	return fmt.Sprintf("%s: %v (raw_len=%d)", e.Purpose, e.Err, len(e.Raw))
}

func (e *ErrUnparseable) Unwrap() error { return e.Err }

// RawResponseFor returns the raw model output carried by err, if any, capped to
// a length that is sensible to store.
func RawResponseFor(err error, limit int) string {
	var u *ErrUnparseable
	if !errors.As(err, &u) {
		return ""
	}
	if limit > 0 && len(u.Raw) > limit {
		return u.Raw[:limit]
	}
	return u.Raw
}

// ExtractPage runs vision on one image and returns verbatim page text.
// Date detection is owned by ExtractMetadata; the date return is always "".
//
// A degenerate transcription (the page repeated over and over) is retried once
// and then trimmed to its first cycle rather than failed, because that first
// cycle is normally a correct read of the page.
func (c *Client) ExtractPage(ctx context.Context, imageBase64 string) (text string, documentDate string, err error) {
	profile := visionPageProfile()
	log.Printf("ollama ExtractPage: model=%q profile=%q endpoint=%s", c.VisionModel, profile.name, visionEndpoint())

	var best string
	for attempt := 1; attempt <= 2; attempt++ {
		raw, stats, err := c.callVision(ctx, profile, imageBase64)
		if err != nil {
			return "", "", err
		}
		parsed := profile.parse(raw)
		rep := analyseRepetition(parsed)
		logCall("vision_page", stats,
			fmt.Sprintf("attempt=%d repeat_ratio=%.2f repeat_period=%d repeat_cycles=%d unreadable=%d",
				attempt, rep.Ratio, rep.Period, rep.Cycles, strings.Count(parsed, "[unleserlich]")))

		if strings.TrimSpace(parsed) == "" {
			return "", "", &ErrEmptyContent{Purpose: "vision page", DoneReason: stats.DoneReason, ThinkingLen: stats.ThinkingLen}
		}
		if !rep.Looped() {
			if isTruncated(stats) {
				// Keep the text: a truncated page is still most of a page,
				// and losing the tail beats losing the document.
				log.Printf("ollama ExtractPage: page truncated at %d tokens; raise OLLAMA_NUM_PREDICT_VISION", stats.EvalTokens)
			}
			return parsed, "", nil
		}
		best = parsed
		log.Printf("ollama ExtractPage: transcription repeated %d cycles of %d lines (attempt %d/2)", rep.Cycles, rep.Period, attempt)
	}

	trimmed := trimToFirstCycle(best, analyseRepetition(best))
	log.Printf("ollama ExtractPage: keeping first cycle only (%d chars from %d)", len(trimmed), len(best))
	return trimmed, "", nil
}

// PageProfileName returns the stable page-extraction profile name for logging
// and extractions.engine_id (same for all vision models).
func (c *Client) PageProfileName() string {
	return visionPageProfile().name
}

// MetadataPromptVersion identifies the prompt set used by ExtractMetadata
// (plain-text translate + structured summary/date). Bumped whenever those
// prompts change in a behavior-relevant way. Exported so the caller can
// persist it on the extractions row.
const MetadataPromptVersion = metadataPromptVersion

// callVision posts one page image to the configured vision endpoint.
//
// /api/chat is the default. Models published with RENDERER/PARSER instead of a
// TEMPLATE — turbo declares RENDERER qwen3-vl-instruct — are templated by
// Ollama's renderer, which drives the chat path. On /api/generate the two are
// not equivalent (ollama#14793), and an instruct model that never receives its
// chat wrapping continues the document rather than answering, which is how a
// page ends up transcribed several times over.
func (c *Client) callVision(ctx context.Context, p pagePromptProfile, imageBase64 string) (string, callStats, error) {
	endpoint := visionEndpoint()
	opts := visionOptions()

	if endpoint == visionEndpointGenerate {
		req := generateRequest{
			Model:   c.VisionModel,
			Prompt:  p.prompt,
			Images:  []string{imageBase64},
			Stream:  false,
			Think:   &noThink,
			Options: opts,
		}
		var resp generateResponse
		if err := c.postJSON(ctx, endpoint, req, &resp); err != nil {
			return "", callStats{Endpoint: endpoint, Model: c.VisionModel}, err
		}
		return resp.Response, resp.stats(endpoint, c.VisionModel), nil
	}

	req := chatRequest{
		Model:   c.VisionModel,
		Stream:  false,
		Think:   &noThink,
		Options: opts,
		Messages: []chatMessage{
			{Role: "user", Content: p.prompt, Images: []string{imageBase64}},
		},
	}
	var resp chatResponse
	if err := c.postJSON(ctx, endpoint, req, &resp); err != nil {
		return "", callStats{Endpoint: endpoint, Model: c.VisionModel}, err
	}
	return resp.Message.Content, resp.stats(endpoint, c.VisionModel), nil
}

// ExtractMetadata translates page by page, summarises the whole English
// document, and extracts document_date from the first page's letterhead.
// If structured fields fail, translation is still returned.
func (c *Client) ExtractMetadata(ctx context.Context, pageParts []string, fullTextOriginal string) (summary, fullTextEnglish, documentDate string, err error) {
	if len(pageParts) == 0 {
		return "", "", "", fmt.Errorf("extract metadata: empty pageParts")
	}

	pageEng, fullTextEnglish, err := c.TranslatePages(ctx, pageParts, fullTextOriginal, nil)
	if err != nil {
		return "", "", "", err
	}

	summary, sumErr := c.SummarizeDocument(ctx, pageEng)
	if sumErr != nil {
		log.Printf("ollama ExtractMetadata: summary failed (%v); continuing", sumErr)
	}

	documentDate, metaErr := c.ExtractDocumentDate(ctx, pageParts[0])
	if metaErr != nil {
		log.Printf("ollama ExtractMetadata: document_date failed (%v); continuing", metaErr)
	}
	return summary, fullTextEnglish, documentDate, nil
}

// TranslateProgress reports page translation progress (1-based done count).
type TranslateProgress func(done, total int)

// TranslatePages translates each page to English (plain text), skipping
// near-blank pages and copying verbatim when the document is already English.
func (c *Client) TranslatePages(ctx context.Context, pageParts []string, fullTextOriginal string, onProgress TranslateProgress) (pageEnglish []string, fullEnglish string, err error) {
	if len(pageParts) == 0 {
		return nil, "", fmt.Errorf("translate pages: empty pageParts")
	}

	pageEnglish = make([]string, len(pageParts))
	if !LikelyNonEnglish(fullTextOriginal) {
		log.Printf("ollama TranslatePages: source looks English; copying verbatim (%d pages)", len(pageParts))
		copy(pageEnglish, pageParts)
		if onProgress != nil {
			onProgress(len(pageParts), len(pageParts))
		}
		return pageEnglish, strings.TrimSpace(strings.Join(pageEnglish, "\n\n")), nil
	}

	for i, page := range pageParts {
		if onProgress != nil {
			onProgress(i, len(pageParts))
		}
		if PageNearBlank(page) {
			log.Printf("ollama TranslatePages: page %d near-blank; skipping LLM", i)
			pageEnglish[i] = strings.TrimSpace(page)
			continue
		}
		eng, err := c.TranslateFullTextEnglish(ctx, page)
		if err != nil {
			return nil, "", fmt.Errorf("translate page %d: %w", i, err)
		}
		pageEnglish[i] = eng
	}
	if onProgress != nil {
		onProgress(len(pageParts), len(pageParts))
	}
	return pageEnglish, strings.TrimSpace(strings.Join(pageEnglish, "\n\n")), nil
}

// PageNearBlank reports whether a page has too little text to bother translating.
func PageNearBlank(page string) bool {
	n := 0
	for _, r := range strings.TrimSpace(page) {
		if r > ' ' {
			n++
			if n >= minPageCharsForTranslate {
				return false
			}
		}
	}
	return true
}

// SummarizeDocument returns a short English summary of the whole document.
// Long multi-page text uses map-reduce (per-page notes, then one final pass).
func (c *Client) SummarizeDocument(ctx context.Context, pageEnglish []string) (string, error) {
	full := strings.TrimSpace(strings.Join(pageEnglish, "\n\n"))
	if full == "" {
		return "", nil
	}
	if len(pageEnglish) <= 1 || len(full) < mapReduceSummaryChars {
		return c.summarizeOnce(ctx, summarizeSystemPrompt, full, "summary")
	}

	var notes []string
	for i, page := range pageEnglish {
		page = strings.TrimSpace(page)
		if PageNearBlank(page) {
			continue
		}
		note, err := c.summarizeOnce(ctx, summarizeMapSystemPrompt, page, "summary_map")
		if err != nil {
			log.Printf("ollama SummarizeDocument: page %d map failed: %v", i, err)
			continue
		}
		if note != "" {
			notes = append(notes, note)
		}
	}
	if len(notes) == 0 {
		return c.summarizeOnce(ctx, summarizeSystemPrompt, full, "summary")
	}
	return c.summarizeOnce(ctx, summarizeSystemPrompt, strings.Join(notes, "\n"), "summary_reduce")
}

func (c *Client) summarizeOnce(ctx context.Context, system, userText, purpose string) (string, error) {
	userContent := "DOCUMENT TEXT:\n" + userText
	req := chatRequest{
		Model:   c.TextModel,
		Stream:  false,
		Think:   &noThink,
		Options: textOptions(len(system) + len(userContent)),
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: userContent},
		},
	}
	var resp chatResponse
	if err := c.postJSON(ctx, "/api/chat", req, &resp); err != nil {
		return "", err
	}
	stats := resp.stats("/api/chat", c.TextModel)
	logCall(purpose, stats)
	content := strings.TrimSpace(stripMarkdownFences(resp.Message.Content))
	if content == "" {
		return "", &ErrEmptyContent{Purpose: purpose, DoneReason: stats.DoneReason, ThinkingLen: stats.ThinkingLen}
	}
	return content, nil
}

// ExtractDocumentDate reads letterhead date from page 1.
func (c *Client) ExtractDocumentDate(ctx context.Context, firstPageOriginal string) (documentDate string, err error) {
	userContent := "DOCUMENT TEXT:\n" + firstPageOriginal
	req := chatRequest{
		Model:   c.TextModel,
		Stream:  false,
		Think:   &noThink,
		Format:  json.RawMessage(documentDateFormat),
		Options: textOptions(len(documentDateSystemPrompt) + len(userContent)),
		Messages: []chatMessage{
			{Role: "system", Content: documentDateSystemPrompt},
			{Role: "user", Content: userContent},
		},
	}
	var resp chatResponse
	if err := c.postJSON(ctx, "/api/chat", req, &resp); err != nil {
		return "", err
	}
	stats := resp.stats("/api/chat", c.TextModel)
	logCall("document_date", stats)
	if strings.TrimSpace(resp.Message.Content) == "" {
		return "", &ErrEmptyContent{Purpose: "document_date", DoneReason: stats.DoneReason, ThinkingLen: stats.ThinkingLen}
	}

	content := resp.Message.Content
	if s := extractJSONBlock(content); s != "" {
		content = s
	}
	content = trimToFirstJSONObject(content)
	content = fixJSONEscapes(content)

	var m struct {
		DocumentDate string `json:"document_date"`
	}
	var rawDate string
	if err := json.Unmarshal([]byte(content), &m); err != nil {
		_, _, dateS := salvageMetadata(content)
		if dateS == "" {
			dateS = salvageStringField(content, "document_date")
		}
		if dateS == "" {
			return "", &ErrUnparseable{Purpose: "parse document_date", Raw: content, Err: err}
		}
		rawDate = dateS
	} else {
		rawDate = m.DocumentDate
	}

	isoDate := coerceISODate(rawDate)
	if rawDate != "" && isoDate == "" {
		log.Printf("ollama document_date: rejecting unparseable document_date=%q", rawDate)
	} else if rawDate != "" && rawDate != isoDate {
		log.Printf("ollama document_date: coerced document_date %q -> %q", rawDate, isoDate)
	}
	if isoDate == "" {
		if fb := letterheadDocumentDate(firstPageOriginal); fb != "" {
			log.Printf("ollama document_date: letterhead Datum/Date fallback %q (model had %q)", fb, rawDate)
			isoDate = fb
		}
	}
	return isoDate, nil
}

// TranslateFullTextEnglish returns an English Markdown translation as plain
// text (no JSON). Retries once on empty content, and again if the reply still
// looks non-English while the source was non-English. When both passes still
// look like failed German, returns empty (fail-closed) — never paraphrased DE.
func (c *Client) TranslateFullTextEnglish(ctx context.Context, fullTextOriginal string) (string, error) {
	log.Printf("ollama TranslateFullTextEnglish: model=%q prompt_version=%s", c.TextModel, TranslateOnlyPromptVersion)

	eng, err := c.translateWithRetry(ctx, translatePlainSystemPrompt, fullTextOriginal)
	if err != nil {
		return "", err
	}
	if LikelyNonEnglish(fullTextOriginal) && LooksPredominantlyGerman(eng) {
		log.Printf("ollama TranslateFullTextEnglish: output still looks non-English; retrying with stronger prompt")
		retry, retryErr := c.translateWithRetry(ctx, translateRetrySystemPrompt, fullTextOriginal)
		if retryErr != nil {
			log.Printf("ollama TranslateFullTextEnglish: strong retry failed: %v; failing closed (empty)", retryErr)
			return "", nil
		}
		// Accept only when the retry is a real translation (not German / not echo).
		if !ShouldRetryTranslation(fullTextOriginal, retry) {
			return retry, nil
		}
		log.Printf("ollama TranslateFullTextEnglish: strong retry still non-English; failing closed (empty)")
		return "", nil
	}
	return eng, nil
}

func (c *Client) translateWithRetry(ctx context.Context, system, fullTextOriginal string) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		eng, stats, err := c.translatePlainOnce(ctx, system, fullTextOriginal)
		if err != nil {
			lastErr = err
			var empty *ErrEmptyContent
			if errors.As(err, &empty) {
				log.Printf("ollama TranslateFullTextEnglish: empty on attempt %d/2", attempt)
				continue
			}
			return "", err
		}
		if isTruncated(stats) {
			log.Printf("ollama TranslateFullTextEnglish: truncated at %d tokens; keeping partial (len=%d)", stats.EvalTokens, len(eng))
		}
		return eng, nil
	}
	return "", lastErr
}

func (c *Client) translatePlainOnce(ctx context.Context, system, fullTextOriginal string) (string, callStats, error) {
	// Pass the letter body alone. Labels like "ORIGINAL DOCUMENT TEXT:" invite
	// OCR-specialist models to rewrite/echo the source instead of translating.
	userContent := strings.TrimSpace(fullTextOriginal)
	req := chatRequest{
		Model:   c.TextModel,
		Stream:  false,
		Think:   &noThink,
		Options: textOptions(len(system) + len(userContent)),
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: userContent},
		},
	}
	var resp chatResponse
	if err := c.postJSON(ctx, "/api/chat", req, &resp); err != nil {
		log.Printf("ollama TranslateFullTextEnglish: post failed: %v", err)
		return "", callStats{Endpoint: "/api/chat", Model: c.TextModel}, err
	}
	stats := resp.stats("/api/chat", c.TextModel)
	logCall("translate", stats)
	content := strings.TrimSpace(stripMarkdownFences(resp.Message.Content))
	if content == "" {
		return "", stats, &ErrEmptyContent{Purpose: "translate", DoneReason: stats.DoneReason, ThinkingLen: stats.ThinkingLen}
	}
	return content, stats, nil
}

// extractStructuredMeta is retained for degrade paths that still want summary+date together.
func (c *Client) extractStructuredMeta(ctx context.Context, sourceText string, multiPage bool) (summary, documentDate string, err error) {
	system := structuredMetaSystemPrompt
	if multiPage {
		system = structuredMetaSystemPrompt + "\n\n" + metadataMultiPageSupplement
	}
	userContent := "DOCUMENT TEXT:\n" + sourceText

	summary, documentDate, err = c.structuredMetaOnce(ctx, system, userContent, json.RawMessage(structuredMetaFormat), "metadata")
	if err == nil {
		return summary, documentDate, nil
	}
	log.Printf("ollama structured meta: combined call failed: %v; trying summary-only", err)

	sumOnly := `{"type":"object","properties":{"summary":{"type":"string"}},"required":["summary"]}`
	summary, _, sumErr := c.structuredMetaOnce(ctx, system, userContent, json.RawMessage(sumOnly), "summary_only")
	if sumErr != nil {
		log.Printf("ollama structured meta: summary-only failed: %v", sumErr)
	}

	dateOnly := `{"type":"object","properties":{"document_date":{"type":"string"}},"required":["document_date"]}`
	_, documentDate, dateErr := c.structuredMetaOnce(ctx, system, userContent, json.RawMessage(dateOnly), "date_only")
	if dateErr != nil {
		log.Printf("ollama structured meta: date-only failed: %v", dateErr)
	}

	if summary == "" && documentDate == "" {
		if err != nil {
			return "", "", err
		}
		return "", "", fmt.Errorf("structured meta: all attempts failed")
	}
	return summary, documentDate, nil
}

func (c *Client) structuredMetaOnce(ctx context.Context, system, userContent string, format json.RawMessage, purpose string) (summary, documentDate string, err error) {
	req := chatRequest{
		Model:   c.TextModel,
		Stream:  false,
		Think:   &noThink,
		Format:  format,
		Options: textOptions(len(system) + len(userContent)),
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: userContent},
		},
	}
	var resp chatResponse
	if err := c.postJSON(ctx, "/api/chat", req, &resp); err != nil {
		return "", "", err
	}
	stats := resp.stats("/api/chat", c.TextModel)
	logCall(purpose, stats)
	if strings.TrimSpace(resp.Message.Content) == "" {
		return "", "", &ErrEmptyContent{Purpose: purpose, DoneReason: stats.DoneReason, ThinkingLen: stats.ThinkingLen}
	}
	if isTruncated(stats) {
		return "", "", truncatedError(purpose, stats)
	}

	content := resp.Message.Content
	if s := extractJSONBlock(content); s != "" {
		content = s
	}
	content = trimToFirstJSONObject(content)
	content = fixJSONEscapes(content)

	var m struct {
		Summary      string `json:"summary"`
		DocumentDate string `json:"document_date"`
	}
	if err := json.Unmarshal([]byte(content), &m); err != nil {
		sumS, _, dateS := salvageMetadata(content)
		if sumS == "" && dateS == "" {
			return "", "", &ErrUnparseable{Purpose: "parse " + purpose, Raw: content, Err: err}
		}
		log.Printf("ollama %s: salvaged via regex", purpose)
		return sumS, coerceISODate(dateS), nil
	}

	rawDate := m.DocumentDate
	isoDate := coerceISODate(rawDate)
	if rawDate != "" && isoDate == "" {
		log.Printf("ollama %s: rejecting unparseable document_date=%q", purpose, rawDate)
	} else if rawDate != "" && rawDate != isoDate {
		log.Printf("ollama %s: coerced document_date %q -> %q", purpose, rawDate, isoDate)
	}
	return strings.TrimSpace(m.Summary), isoDate, nil
}

// fixJSONEscapes fixes invalid JSON escape sequences (e.g. \B) that some LLMs emit.
// Valid JSON escapes after \ are: " \ / b f n r t u (and \uXXXX). Others become \\ + char.
func fixJSONEscapes(s string) string {
	const validEscapes = `"\/bfnrtu`
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			next := s[i+1]
			if next == 'u' {
				// \uXXXX - leave as is (valid)
				b.WriteByte(c)
				continue
			}
			if strings.IndexByte(validEscapes, next) >= 0 {
				b.WriteByte(c)
				continue
			}
			// Invalid escape: \X -> \\X so decoder sees literal backslash + X
			b.WriteByte('\\')
			b.WriteByte('\\')
			b.WriteByte(next)
			i++
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// trimToFirstJSONObject returns the substring from the first { to its matching }.
// This avoids "invalid character 'm' after object" when the model appends text after the JSON.
func trimToFirstJSONObject(s string) string {
	s = strings.TrimSpace(s)
	start := strings.Index(s, "{")
	if start < 0 {
		return s
	}
	depth := 0
	inString := false
	escape := false
	var quote byte
	for i := start; i < len(s); i++ {
		c := s[i]
		if escape {
			escape = false
			continue
		}
		if inString {
			if c == '\\' && quote == '"' {
				escape = true
				continue
			}
			if c == quote {
				inString = false
			}
			continue
		}
		switch c {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		case '"', '\'':
			inString = true
			quote = c
		}
	}
	return s
}

func extractJSONBlock(s string) string {
	const start = "```json"
	const end = "```"
	s = strings.TrimSpace(s)
	i := strings.Index(s, start)
	if i < 0 {
		return ""
	}
	s = s[i+len(start):]
	j := strings.Index(s, end)
	if j < 0 {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(s[:j])
}

func (c *Client) postJSON(ctx context.Context, path string, body any, result any) error {
	u := strings.TrimSuffix(c.BaseURL, "/") + path
	enc, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(enc))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		log.Printf("ollama postJSON %s: %v", path, err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("ollama postJSON %s: %s %s", path, resp.Status, string(body))
		return fmt.Errorf("ollama %s: %s", resp.Status, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(result)
}
