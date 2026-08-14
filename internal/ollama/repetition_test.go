package ollama

import (
	"strings"
	"testing"
)

func TestAnalyseRepetition_CleanPageNotFlagged(t *testing.T) {
	got := analyseRepetition(syntheticGermanPage)
	if got.Looped() {
		t.Fatalf("clean page flagged as looped: period=%d cycles=%d ratio=%.2f", got.Period, got.Cycles, got.Ratio)
	}
}

func TestAnalyseRepetition_DetectsTripledPage(t *testing.T) {
	looped := syntheticGermanPage + "\n\n" + syntheticGermanPage + "\n\n" + syntheticGermanPage
	got := analyseRepetition(looped)
	if !got.Looped() {
		t.Fatalf("tripled page not detected: ratio=%.2f", got.Ratio)
	}
	if got.Cycles != 3 {
		t.Errorf("cycles = %d, want 3", got.Cycles)
	}
	if got.Ratio < 0.6 {
		t.Errorf("ratio = %.2f, want >= 0.6 for a tripled page", got.Ratio)
	}
}

func TestTrimToFirstCycle_RecoversOneCopy(t *testing.T) {
	looped := syntheticGermanPage + "\n\n" + syntheticGermanPage
	trimmed := trimToFirstCycle(looped, analyseRepetition(looped))

	if analyseRepetition(trimmed).Looped() {
		t.Error("trimmed text still looks looped")
	}
	// The first cycle must survive intact: check distinctive tokens plus
	// the ß that OCR most often loses.
	for _, want := range []string{"12 345 678 901", "10.07.2026", "Weißstr. 14"} {
		if !strings.Contains(trimmed, want) {
			t.Errorf("trimmed text lost %q", want)
		}
	}
	if n := strings.Count(trimmed, "Erteilung einer Berechtigung"); n != 1 {
		t.Errorf("subject line appears %d times, want 1", n)
	}
}

func TestTrimToFirstCycle_LeavesCleanTextAlone(t *testing.T) {
	if got := trimToFirstCycle(syntheticGermanPage, analyseRepetition(syntheticGermanPage)); got != syntheticGermanPage {
		t.Error("clean text was modified")
	}
}

func TestAnalyseRepetition_ShortTextIsNotACycle(t *testing.T) {
	// Two identical substantial lines are normal in real documents (repeated
	// table cells, address blocks) and must not trip the detector.
	line := "Gültig für: Veranlagungszeitraum unbefristet\n"
	if got := analyseRepetition(line + line); got.Looped() {
		t.Errorf("two identical lines flagged as a loop: period=%d", got.Period)
	}
}
