package ollama

import (
	"testing"
)

func TestNormalizeForTranslationCompare(t *testing.T) {
	a := "a\r\nb"
	b := "a\nb"
	if NormalizeForTranslationCompare(a) != NormalizeForTranslationCompare(b) {
		t.Fatalf("CRLF vs LF should match")
	}
}

func TestShouldRetryTranslation(t *testing.T) {
	de := "Sehr geehrte Damen und Herren, hier ist Ihre Rechnung für 100 EUR."
	en := "Dear Sir or Madam, here is your invoice for 100 EUR."
	if ShouldRetryTranslation(de, en) {
		t.Fatal("different texts should not retry")
	}
	if !ShouldRetryTranslation(de, de) {
		t.Fatal("identical German text should retry")
	}
	if ShouldRetryTranslation(en, en) {
		t.Fatal("identical English should not retry")
	}
	longDE := "Dies ist ein langes deutsches Schreiben und der Betrag ist hoch genug für das Wortmuster."
	if !ShouldRetryTranslation(longDE, longDE) {
		t.Fatal("long German with function words should retry when echoed")
	}
	// Slightly rephrased German still counts as a failed translation.
	almost := "Sehr geehrte Damen und Herren — hier ist Ihre Rechnung für 100 EUR."
	if !ShouldRetryTranslation(de, almost) {
		t.Fatal("German-looking output that is not an exact echo should still retry")
	}
}

func TestLooksPredominantlyGerman_UmlautAloneIsNotEnough(t *testing.T) {
	// Correct English that keeps a German street name must not look "German".
	en := "Lindenstadt Civic Office\n\nWeißstr. 14 12345 Lindenstadt\n\nDear Sir or Madam, please find your invoice for the tax amount."
	if LooksPredominantlyGerman(en) {
		t.Fatal("English with Weißstr. must not look predominantly German")
	}
	if ShouldRetryTranslation(
		"Sehr geehrte Damen und Herren, hier ist Ihre Rechnung für die Steuer.",
		en,
	) {
		t.Fatal("ShouldRetryTranslation must not fire on English with umlaut street name")
	}
}

func TestLikelyNonEnglish_SyntheticReferencePair(t *testing.T) {
	if !LikelyNonEnglish(syntheticGermanPage) {
		t.Fatal("synthetic German source should be LikelyNonEnglish")
	}
	if LooksPredominantlyGerman(syntheticEnglishPage) {
		t.Fatal("synthetic English must not look predominantly German (ß in street name is OK)")
	}
	if ShouldRetryTranslation(syntheticGermanPage, syntheticEnglishPage) {
		t.Fatal("synthetic de→en pair must not trigger ShouldRetryTranslation")
	}
	if ShouldOuterTranslateRetry(syntheticGermanPage, syntheticEnglishPage) {
		t.Fatal("synthetic de→en must not trigger outer retry")
	}
}

func TestShouldOuterTranslateRetry_OnlyEcho(t *testing.T) {
	de := "Sehr geehrte Damen und Herren, hier ist Ihre Rechnung für 100 EUR."
	en := "Dear Sir or Madam, here is your invoice for 100 EUR."
	if ShouldOuterTranslateRetry(de, en) {
		t.Fatal("good English must not outer-retry")
	}
	if !ShouldOuterTranslateRetry(de, de) {
		t.Fatal("echo must outer-retry")
	}
	// Empty after TranslateFullTextEnglish means fail-closed / exhausted empties —
	// outer must not pay another full document translate.
	if ShouldOuterTranslateRetry(de, "   ") {
		t.Fatal("empty must not outer-retry")
	}
	// German-looking but non-echo: inner TranslateFullTextEnglish already
	// ran the strong prompt; outer must not repeat that work.
	almost := "Sehr geehrte Damen und Herren — hier ist Ihre Rechnung für 100 EUR."
	if ShouldOuterTranslateRetry(de, almost) {
		t.Fatal("non-echo German-looking output is inner-only; outer must skip")
	}
}
