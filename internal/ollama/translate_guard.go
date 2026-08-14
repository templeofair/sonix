package ollama

import (
	"regexp"
	"strings"
)

// NormalizeForTranslationCompare normalizes line endings and trims space so
// we can compare "same text" without trivial OCR/layout differences.
func NormalizeForTranslationCompare(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.TrimSpace(s)
}

// TranslationEchoesOriginal reports whether eng is effectively the same as orig.
func TranslationEchoesOriginal(orig, eng string) bool {
	if strings.TrimSpace(orig) == "" {
		return false
	}
	return NormalizeForTranslationCompare(orig) == NormalizeForTranslationCompare(eng)
}

// German and English function-word cues. Umlauts alone are not enough: correct
// English translations of German letters keep street names like "Weißstr.".
var germanFunctionWordRE = regexp.MustCompile(`(?i)\b(der|die|das|und|nicht|mit|für|ist|sind|sehr|geehrte|ihre|rechnung|betrag|bitte|diese|dieser|dieses|einen|einer|einem|sowie|oder|auch|noch|wird|wurde|haben|hat|hier|vom|zum|zur|über|zwischen)\b`)

var englishFunctionWordRE = regexp.MustCompile(`(?i)\b(the|and|for|with|is|are|this|that|your|please|dear|invoice|from|have|has|will|can|not|you|we|our|was|were|been|their|they|which|would|should|could|about|into|than|then|also|only|other|more|some|what|when|where|who|how|all|each|every|both|many|most|just|over|after|before|office|tax|amount|date)\b`)

func umlautCount(s string) int {
	n := 0
	for _, r := range s {
		switch r {
		case 'ß', 'ä', 'ö', 'ü', 'Ä', 'Ö', 'Ü':
			n++
		}
	}
	return n
}

func cueCount(s string, re *regexp.Regexp) int {
	return len(re.FindAllString(s, -1))
}

// LooksPredominantlyGerman reports whether German function-word cues dominate
// English ones. Sparse umlauts in otherwise-English text (addresses, names)
// do not count as German.
func LooksPredominantlyGerman(s string) bool {
	g := cueCount(s, germanFunctionWordRE)
	e := cueCount(s, englishFunctionWordRE)
	if g == 0 && e == 0 {
		// No function words either way: only treat dense umlauts as German.
		return umlautCount(s) >= 3 && len([]rune(s)) >= 40
	}
	return g > e
}

// LikelyNonEnglish is used to decide whether a source document should be
// translated. Short snippets with umlauts still count (letterhead fragments);
// longer text uses cue dominance so English with "Weißstr." is not forced
// through the translate path.
func LikelyNonEnglish(orig string) bool {
	s := strings.TrimSpace(orig)
	if s == "" {
		return false
	}
	if LooksPredominantlyGerman(s) {
		return true
	}
	// Short fragments: a single umlaut is a useful OCR/language hint.
	if len([]rune(s)) < 40 && umlautCount(s) > 0 {
		return true
	}
	return false
}

// ShouldRetryTranslation is true when the model failed to produce English:
// empty output, an echo of the source, or output that still looks predominantly
// German (not merely containing an umlaut in a proper noun).
func ShouldRetryTranslation(orig, eng string) bool {
	if !LikelyNonEnglish(orig) {
		return false
	}
	if strings.TrimSpace(eng) == "" {
		return true
	}
	if TranslationEchoesOriginal(orig, eng) {
		return true
	}
	return LooksPredominantlyGerman(eng)
}

// ShouldOuterTranslateRetry is for the post-pipeline retry after
// TranslateFullTextEnglish (which already empty-retries and strong-retries).
// Empty after that means fail-closed — do not pay another full document call.
// Exact echo is the only remaining case worth one outer attempt.
func ShouldOuterTranslateRetry(orig, eng string) bool {
	if !LikelyNonEnglish(orig) {
		return false
	}
	if strings.TrimSpace(eng) == "" {
		return false
	}
	return TranslationEchoesOriginal(orig, eng)
}
