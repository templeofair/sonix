package ollama

import (
	"strings"
	"testing"
)

func TestVisionPageProfile_Unified(t *testing.T) {
	p := visionPageProfile()
	if p.name != UnifiedVisionProfileName {
		t.Errorf("name = %q, want %q", p.name, UnifiedVisionProfileName)
	}
	if p.prompt != VisionPageExtractPrompt {
		t.Errorf("prompt = %q, want %q", p.prompt, VisionPageExtractPrompt)
	}
	if p.parse == nil {
		t.Fatal("parse is nil")
	}
}

func TestStripMarkdownFences(t *testing.T) {
	cases := map[string]string{
		"plain text":                     "plain text",
		"```\nhello\n```":                "hello",
		"```markdown\nhello\nworld\n```": "hello\nworld",
		"```\nhello\n":                   "hello", // no closing fence: open fence dropped, rest preserved.
		"```":                            "",      // lone fence, no newline → stripped prefix leaves empty.
		"no fence \n here":               "no fence \n here",
	}
	for in, want := range cases {
		if got := stripMarkdownFences(in); got != want {
			t.Errorf("stripMarkdownFences(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSalvageMetadata_TruncatedTrailingBackslash(t *testing.T) {
	// Truncated JSON: the model hallucinated empty markdown rows until it
	// hit its context limit, ending mid-string with a trailing unescaped \.
	// Strict json.Unmarshal rejects this; salvage should still recover the
	// english text and stay calm on the missing summary/date.
	raw := `{
  "full_text_english": "Nordbahn AG\nBeispiel-Allee 1-3\n12345 Lindenstadt\n| 19.12.24 | Einzelticket | 4,40 € |\n|  |  |  |\`
	sum, eng, date := salvageMetadata(raw)
	if sum != "" {
		t.Errorf("summary should be empty (model never reached it), got %q", sum)
	}
	if date != "" {
		t.Errorf("date should be empty (no date field in truncated output), got %q", date)
	}
	if eng == "" {
		t.Fatalf("english should be salvaged from truncated value, got empty")
	}
	// Spot-check the salvaged content: first line + a known table row.
	if !strings.Contains(eng, "Nordbahn AG") {
		t.Errorf("salvaged english missing first line: %q", eng)
	}
	if !strings.Contains(eng, "Einzelticket | 4,40 €") {
		t.Errorf("salvaged english missing table row: %q", eng)
	}
	// Newlines must be decoded from \n escapes.
	if !strings.Contains(eng, "\n") {
		t.Errorf("salvaged english should contain decoded newlines: %q", eng)
	}
}

func TestSalvageMetadata_AllFields(t *testing.T) {
	raw := `{"full_text_english": "Hello world.\nSecond line.", "summary": "A test document.", "document_date": "2024-12-19"}`
	sum, eng, date := salvageMetadata(raw)
	if sum != "A test document." {
		t.Errorf("summary = %q, want %q", sum, "A test document.")
	}
	if eng != "Hello world.\nSecond line." {
		t.Errorf("english = %q, want %q", eng, "Hello world.\nSecond line.")
	}
	if date != "2024-12-19" {
		t.Errorf("date = %q, want %q", date, "2024-12-19")
	}
}

func TestSalvageMetadata_EscapedQuotes(t *testing.T) {
	raw := `{"summary": "He said \"hello\" loudly.", "document_date": "2024-01-15"}`
	sum, _, date := salvageMetadata(raw)
	if sum != `He said "hello" loudly.` {
		t.Errorf("summary = %q, want %q", sum, `He said "hello" loudly.`)
	}
	if date != "2024-01-15" {
		t.Errorf("date = %q, want %q", date, "2024-01-15")
	}
}

func TestSalvageMetadata_NoFields(t *testing.T) {
	// Nothing recoverable: caller should treat this as a real failure.
	raw := `garbage that is not even close to JSON`
	sum, eng, date := salvageMetadata(raw)
	if sum != "" || eng != "" || date != "" {
		t.Errorf("expected all empty, got sum=%q eng=%q date=%q", sum, eng, date)
	}
}

func TestSalvageMetadata_DateMustBeISO(t *testing.T) {
	// Don't accept "December 19, 2024" or "19.12.2024" — only strict ISO.
	raw := `{"document_date": "December 19, 2024"}`
	if _, _, date := salvageMetadata(raw); date != "" {
		t.Errorf("non-ISO date should not match, got %q", date)
	}
	raw2 := `{"document_date": "19.12.2024"}`
	if _, _, date := salvageMetadata(raw2); date != "" {
		t.Errorf("German-format date should not match, got %q", date)
	}
}

func TestCoerceISODate(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"already ISO", "2024-12-19", "2024-12-19"},
		{"ISO unpadded month/day (model quirk)", "2026-7-10", "2026-07-10"},
		{"German long DD.MM.YYYY", "19.12.2024", "2024-12-19"},
		{"German long single-digit day/month", "5.3.2024", "2024-03-05"},
		{"German short DD.MM.YY", "19.12.24", "2024-12-19"},
		{"slash format DD/MM/YYYY", "19/12/2024", "2024-12-19"},
		{"trim whitespace then ISO", "  2024-01-15  ", "2024-01-15"},
		{"empty stays empty", "", ""},
		{"unparseable English month name", "December 19, 2024", ""},
		{"month out of range", "19.13.2024", ""},
		{"day out of range", "32.12.2024", ""},
		{"only year", "2024", ""},
		{"reversed already-ISO-like", "12-19-2024", ""},
		{"alpha junk", "N/A", ""},
		{"single digit short year normalized to 20YY", "1.1.24", "2024-01-01"},
		{"trailing junk after German date", "10.07.2026.", ""},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			if got := coerceISODate(c.in); got != c.want {
				t.Errorf("coerceISODate(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestLetterheadDocumentDate_DemoStyle(t *testing.T) {
	got := letterheadDocumentDate(syntheticLetterhead)
	if got != "2026-07-10" {
		t.Fatalf("letterheadDocumentDate = %q, want 2026-07-10 (not Freischaltung)", got)
	}
}

func TestLetterheadDocumentDate_IgnoresBodyOnly(t *testing.T) {
	page := "Sehr geehrte Damen und Herren,\nFreischaltung bis: 05.10.2026\n"
	if got := letterheadDocumentDate(page); got != "" {
		t.Fatalf("expected empty without Datum: line, got %q", got)
	}
}

func TestCoerceISODate_DemoPayloadLength(t *testing.T) {
	// content_len on a Demo document_date call matches this payload.
	payload := `{"document_date":"2026-7-10"}`
	if len(payload) != 29 {
		t.Fatalf("fixture length %d, want 29", len(payload))
	}
	if got := coerceISODate("2026-7-10"); got != "2026-07-10" {
		t.Fatalf("coerceISODate = %q, want 2026-07-10", got)
	}
}

func TestParseGermanOCRResponse(t *testing.T) {
	cases := map[string]string{
		"Hallo Welt":              "Hallo Welt",
		"  Hallo Welt  ":          "Hallo Welt",
		"```markdown\nHallo\n```": "Hallo",
		"```\nHallo\nWelt\n```":   "Hallo\nWelt",
	}
	for in, want := range cases {
		if got := parseGermanOCRResponse(in); got != want {
			t.Errorf("parse(%q) = %q, want %q", in, got, want)
		}
	}
}
