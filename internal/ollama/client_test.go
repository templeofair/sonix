package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// capturedRequest is the decoded body of one call to the fake Ollama.
type capturedRequest struct {
	path string
	body map[string]any
}

// fakeOllama serves canned replies and records what was sent. replies are
// consumed in order; the last one repeats if there are more calls than replies.
func fakeOllama(t *testing.T, replies ...any) (*Client, *[]capturedRequest) {
	t.Helper()
	var got []capturedRequest
	call := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		got = append(got, capturedRequest{path: r.URL.Path, body: body})

		if len(replies) == 0 {
			t.Errorf("unexpected Ollama call to %s", r.URL.Path)
			http.Error(w, "unexpected call", http.StatusInternalServerError)
			return
		}
		i := call
		if i >= len(replies) {
			i = len(replies) - 1
		}
		call++
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(replies[i]); err != nil {
			t.Errorf("encode reply: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	return NewClient(srv.URL, "vision-model", "text-model"), &got
}

// chatReply builds an /api/chat response body.
func chatReply(content string, extra map[string]any) map[string]any {
	m := map[string]any{
		"message":           map[string]any{"content": content},
		"done_reason":       "stop",
		"prompt_eval_count": 120,
		"eval_count":        340,
	}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func TestThinkIsTopLevelAndFalse(t *testing.T) {
	// think must never live inside options: Ollama ignores unrecognized
	// option keys silently, so options:{think:false} is a no-op that looks
	// correct (ollama#14793). On a thinking model that no-op is what empties
	// message.content and produces "unexpected end of JSON input".
	cases := map[string]struct {
		replies []any
		call    func(*Client) error
	}{
		"vision": {
			replies: []any{chatReply(syntheticGermanPage, nil)},
			call: func(c *Client) error {
				_, _, err := c.ExtractPage(context.Background(), "aGk=")
				return err
			},
		},
		"metadata": {
			replies: []any{
				chatReply("Invoice for services.", nil),
				chatReply("This is a short English summary.", nil),
				chatReply(`{"document_date":"2026-07-10"}`, nil),
			},
			call: func(c *Client) error {
				_, _, _, err := c.ExtractMetadata(context.Background(), []string{"Rechnung"}, "Rechnung")
				return err
			},
		},
		"translate_only": {
			replies: []any{chatReply("Invoice for services.", nil)},
			call: func(c *Client) error {
				_, err := c.TranslateFullTextEnglish(context.Background(), "Rechnung")
				return err
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			client, got := fakeOllama(t, tc.replies...)
			if err := tc.call(client); err != nil {
				t.Fatalf("call failed: %v", err)
			}
			req := (*got)[0]

			think, ok := req.body["think"]
			if !ok {
				t.Fatal("think missing from top level")
			}
			if think != false {
				t.Errorf("think = %v, want false", think)
			}
			if opts, ok := req.body["options"].(map[string]any); ok {
				if _, bad := opts["think"]; bad {
					t.Error("think must not be inside options; Ollama ignores it there")
				}
			}
		})
	}
}

func TestVisionUsesChatEndpointWithImageOnMessage(t *testing.T) {
	// Renderer-based models (turbo declares RENDERER qwen3-vl-instruct) are
	// templated by Ollama's renderer, which drives /api/chat. Without that
	// wrapping an instruct model continues the document instead of
	// transcribing it, producing the repetition loop.
	client, got := fakeOllama(t, chatReply(syntheticGermanPage, nil))
	if _, _, err := client.ExtractPage(context.Background(), "aGk="); err != nil {
		t.Fatalf("ExtractPage: %v", err)
	}

	req := (*got)[0]
	if req.path != "/api/chat" {
		t.Fatalf("endpoint = %s, want /api/chat", req.path)
	}
	msgs, ok := req.body["messages"].([]any)
	if !ok || len(msgs) != 1 {
		t.Fatalf("messages = %v, want exactly one user message", req.body["messages"])
	}
	msg := msgs[0].(map[string]any)
	if msg["role"] != "user" {
		t.Errorf("role = %v, want user", msg["role"])
	}
	if imgs, ok := msg["images"].([]any); !ok || len(imgs) != 1 {
		t.Errorf("images = %v, want one image on the user message", msg["images"])
	}
	// No system message: turbo's own Modelfile SYSTEM prompt carries the
	// transcription rules, and a request system message would replace it.
	if _, bad := msg["system"]; bad {
		t.Error("vision call must not send a system message")
	}
}

func TestVisionEndpointOverride(t *testing.T) {
	t.Setenv("OLLAMA_VISION_ENDPOINT", "/api/generate")
	client, got := fakeOllama(t, map[string]any{"response": syntheticGermanPage, "done_reason": "stop"})
	if _, _, err := client.ExtractPage(context.Background(), "aGk="); err != nil {
		t.Fatalf("ExtractPage: %v", err)
	}
	if p := (*got)[0].path; p != "/api/generate" {
		t.Errorf("endpoint = %s, want /api/generate when overridden", p)
	}
}

func TestEveryCallStatesItsBudget(t *testing.T) {
	client, got := fakeOllama(t,
		chatReply("Invoice body in English.", nil),
		chatReply("Short summary.", nil),
		chatReply(`{"document_date":""}`, nil),
	)
	if _, _, _, err := client.ExtractMetadata(context.Background(), []string{"Rechnung"}, strings.Repeat("Rechnung ", 500)); err != nil {
		t.Fatalf("ExtractMetadata: %v", err)
	}
	if len(*got) < 1 {
		t.Fatal("no requests captured")
	}
	if _, hasFormat := (*got)[0].body["format"]; hasFormat {
		t.Error("translate call must not send format/json envelope")
	}
	opts, ok := (*got)[0].body["options"].(map[string]any)
	if !ok {
		t.Fatal("options missing")
	}
	for _, key := range []string{"num_ctx", "num_predict", "temperature", "top_k", "repeat_penalty"} {
		if _, ok := opts[key]; !ok {
			t.Errorf("options missing %s", key)
		}
	}
}

func TestExtractMetadata_PagePipeline(t *testing.T) {
	client, got := fakeOllama(t,
		chatReply("English page.", nil),
		chatReply("Summary of the letter.", nil),
		chatReply(`{"document_date":"2026-07-10"}`, nil),
	)
	sum, eng, date, err := client.ExtractMetadata(context.Background(), []string{syntheticGermanPage}, syntheticGermanPage)
	if err != nil {
		t.Fatalf("ExtractMetadata: %v", err)
	}
	if eng != "English page." {
		t.Errorf("english = %q", eng)
	}
	if sum != "Summary of the letter." {
		t.Errorf("summary = %q", sum)
	}
	if date != "2026-07-10" {
		t.Errorf("date=%q", date)
	}
	if len(*got) != 3 {
		t.Errorf("calls = %d, want translate+summary+document_date", len(*got))
	}
}

func TestExtractDocumentDate_UnpaddedISO(t *testing.T) {
	client, _ := fakeOllama(t, chatReply(`{"document_date":"2026-7-10"}`, nil))
	page := "Datum: 10.07.2026\nFreischaltung bis: 05.10.2026\n"
	date, err := client.ExtractDocumentDate(context.Background(), page)
	if err != nil {
		t.Fatalf("ExtractDocumentDate: %v", err)
	}
	if date != "2026-07-10" {
		t.Fatalf("date=%q, want 2026-07-10", date)
	}
}

func TestExtractDocumentDate_LetterheadFallback(t *testing.T) {
	client, _ := fakeOllama(t, chatReply(`{"document_date":""}`, nil))
	page := "Bürgeramt Lindenstadt\n\nDatum: 10.07.2026\n\nFreischaltung bis: 05.10.2026\n"
	date, err := client.ExtractDocumentDate(context.Background(), page)
	if err != nil {
		t.Fatalf("ExtractDocumentDate: %v", err)
	}
	if date != "2026-07-10" {
		t.Fatalf("date=%q, want letterhead fallback 2026-07-10", date)
	}
}

// germanLetterBody is long enough for cue-based German detection (function words).
const germanLetterBody = "Sehr geehrte Damen und Herren, hier ist Ihre Rechnung und der Betrag ist nicht klein. Bitte prüfen Sie diese Unterlagen sowie die Frist."

// paraphrasedGermanLetter differs from germanLetterBody but is still predominantly German.
const paraphrasedGermanLetter = "Sehr geehrte Damen und Herren — anbei die Rechnung und der Betrag für diese Sache. Bitte prüfen Sie auch die Unterlagen sowie die Frist."

// englishLetterBody is a clear English translation (cue-dominant English).
const englishLetterBody = "Dear Sir or Madam, here is your invoice and the amount is not small. Please check these documents as well as the deadline."

func TestTranslateFullTextEnglish_RejectsParaphrasedGerman(t *testing.T) {
	// Characterization (Phase 1): strong-retry accept OR currently returns paraphrased
	// German as success. Desired: empty or error — never ship German as english.
	if !LooksPredominantlyGerman(germanLetterBody) || !LooksPredominantlyGerman(paraphrasedGermanLetter) {
		t.Fatal("fixture must look predominantly German")
	}
	if TranslationEchoesOriginal(germanLetterBody, paraphrasedGermanLetter) {
		t.Fatal("paraphrase fixture must not be an exact echo")
	}

	client, got := fakeOllama(t,
		chatReply(paraphrasedGermanLetter, nil),
		chatReply(paraphrasedGermanLetter+" Noch ein Satz mit der und die.", nil),
	)
	eng, err := client.TranslateFullTextEnglish(context.Background(), germanLetterBody)
	if len(*got) < 2 {
		t.Fatalf("expected first pass + strong retry, got %d calls", len(*got))
	}
	if err == nil && strings.TrimSpace(eng) != "" && LooksPredominantlyGerman(eng) {
		t.Fatalf("accepted paraphrased German as english (len=%d); want empty or error", len(eng))
	}
}

func TestTranslateFullTextEnglish_AcceptsEnglishOnStrongRetry(t *testing.T) {
	client, got := fakeOllama(t,
		chatReply(paraphrasedGermanLetter, nil),
		chatReply(englishLetterBody, nil),
	)
	eng, err := client.TranslateFullTextEnglish(context.Background(), germanLetterBody)
	if err != nil {
		t.Fatalf("TranslateFullTextEnglish: %v", err)
	}
	if len(*got) < 2 {
		t.Fatalf("expected first pass + strong retry, got %d calls", len(*got))
	}
	if LooksPredominantlyGerman(eng) {
		t.Fatalf("english retry was rejected or overwritten: %q", eng)
	}
	if eng != englishLetterBody {
		t.Fatalf("eng = %q, want English strong-retry body", eng)
	}
}

func TestTranslatePages_SkipsEnglishAndBlank(t *testing.T) {
	client, got := fakeOllama(t) // should not be called
	pages := []string{"Hello world, this is already English text for a letter.", "   \n  "}
	out, full, err := client.TranslatePages(context.Background(), pages, pages[0], nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(*got) != 0 {
		t.Fatalf("expected no LLM calls for English source, got %d", len(*got))
	}
	if out[0] != pages[0] {
		t.Errorf("english page not copied")
	}
	if full == "" {
		t.Error("full empty")
	}
}

func TestPageNearBlank(t *testing.T) {
	if !PageNearBlank("hi") {
		t.Error("short page should be near blank")
	}
	if PageNearBlank(syntheticGermanPage) {
		t.Error("demo page should not be near blank")
	}
}

func TestEmptyContentIsItsOwnError(t *testing.T) {
	reply := chatReply("", map[string]any{
		"message":     map[string]any{"content": "", "thinking": strings.Repeat("Ich denke nach. ", 200)},
		"done_reason": "stop",
	})
	client, _ := fakeOllama(t, reply)

	_, err := client.TranslateFullTextEnglish(context.Background(), "Rechnung")
	var empty *ErrEmptyContent
	if !errors.As(err, &empty) {
		t.Fatalf("err = %v (%T), want *ErrEmptyContent", err, err)
	}
	if empty.ThinkingLen == 0 {
		t.Error("thinking length not captured")
	}
}

func TestPlainTranslateHasNoJSONFormat(t *testing.T) {
	client, got := fakeOllama(t, chatReply("# Dear Sir\n\nPlease find enclosed.", nil))
	src := "Sehr geehrter Herr"
	eng, err := client.TranslateFullTextEnglish(context.Background(), src)
	if err != nil {
		t.Fatalf("TranslateFullTextEnglish: %v", err)
	}
	if !strings.Contains(eng, "Dear Sir") {
		t.Errorf("english = %q", eng)
	}
	if _, has := (*got)[0].body["format"]; has {
		t.Error("plain translate must not use format")
	}
	msgs, _ := (*got)[0].body["messages"].([]any)
	if len(msgs) < 2 {
		t.Fatalf("want system+user messages, got %d", len(msgs))
	}
	user, _ := msgs[1].(map[string]any)
	content, _ := user["content"].(string)
	if content != src {
		t.Errorf("user content = %q, want letter body only (no ORIGINAL DOCUMENT TEXT label)", content)
	}
	if strings.Contains(content, "ORIGINAL DOCUMENT TEXT") {
		t.Error("translate user payload must not use ORIGINAL DOCUMENT TEXT label")
	}
}

func TestVisionLoopRetriesThenKeepsFirstCycle(t *testing.T) {
	// Both attempts loop, so we salvage the first cycle instead of failing:
	// that cycle is normally a correct read of the page.
	looped := syntheticGermanPage + "\n\n" + syntheticGermanPage + "\n\n" + syntheticGermanPage
	client, got := fakeOllama(t, chatReply(looped, nil))

	text, _, err := client.ExtractPage(context.Background(), "aGk=")
	if err != nil {
		t.Fatalf("ExtractPage: %v", err)
	}
	if len(*got) != 2 {
		t.Errorf("made %d calls, want 2 (one retry)", len(*got))
	}
	if analyseRepetition(text).Looped() {
		t.Error("returned text is still looped")
	}
	if !strings.Contains(text, "12 345 678 901") {
		t.Error("salvaged text lost the reference number")
	}
}

func TestVisionCleanPageIsNotRetried(t *testing.T) {
	client, got := fakeOllama(t, chatReply(syntheticGermanPage, nil))
	if _, _, err := client.ExtractPage(context.Background(), "aGk="); err != nil {
		t.Fatalf("ExtractPage: %v", err)
	}
	if len(*got) != 1 {
		t.Errorf("made %d calls, want 1 for clean output", len(*got))
	}
}

func TestVisionEmptyPageFails(t *testing.T) {
	client, _ := fakeOllama(t, chatReply("   ", nil))
	_, _, err := client.ExtractPage(context.Background(), "aGk=")
	var empty *ErrEmptyContent
	if !errors.As(err, &empty) {
		t.Fatalf("err = %v (%T), want *ErrEmptyContent", err, err)
	}
}
