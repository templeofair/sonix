package ollama

import (
	"strings"
)

// Repetition detection for page transcription.
//
// A vision model that is not templated as an instruct model treats the page as
// something to continue rather than a question to answer, and transcribes it
// over and over. The output stays valid Markdown, so nothing downstream
// notices: the translation is then made from a tripled document and the
// summary describes it. repeat_penalty does not help, because it only looks
// back over repeat_last_n tokens (64 by default) and a page repeat is far
// beyond that window.
//
// Detecting it is cheap and gives us both a bench metric and a recovery: the
// first cycle is usually a correct transcription, so trimming to it salvages
// the page instead of failing it.

const (
	// minSignificantLine ignores short lines (single numbers, stray pipes,
	// blank separators) which repeat legitimately in any document.
	minSignificantLine = 20
	// minPeriodLines requires a repeating unit of real substance. Two
	// identical lines are normal; three or more consecutive lines repeating
	// as a block is not.
	minPeriodLines = 3
	// periodMatchThreshold allows for a truncated final cycle and small
	// sampling differences between cycles.
	periodMatchThreshold = 0.9
)

// repetitionReport describes how much of a transcription is a repeat of
// itself. Ratio is reported even when no exact cycle is found, so partial
// degeneration still shows up in telemetry.
type repetitionReport struct {
	// Ratio is the fraction of significant lines that duplicate an earlier
	// line: 0 for clean output, approaching 1 for a heavy loop.
	Ratio float64
	// Period is the number of significant lines in the repeating unit, or 0
	// when no cycle was detected.
	Period int
	// Cycles is how many times the unit repeats (2 or more when Period > 0).
	Cycles int
}

// Looped reports whether the output should be treated as degenerate.
func (r repetitionReport) Looped() bool { return r.Period > 0 }

// analyseRepetition looks for a repeating cycle of lines in text.
func analyseRepetition(text string) repetitionReport {
	lines := strings.Split(text, "\n")
	_, norm := significantLines(lines)
	report := repetitionReport{Ratio: duplicateRatio(norm)}
	if len(norm) < minPeriodLines*2 {
		return report
	}
	for period := minPeriodLines; period <= len(norm)/2; period++ {
		if periodScore(norm, period) >= periodMatchThreshold {
			report.Period = period
			report.Cycles = len(norm) / period
			// Trust the cycle over the line-level ratio: a document whose
			// cycles differ slightly still repeats in full.
			report.Ratio = 1 - float64(period)/float64(len(norm))
			return report
		}
	}
	return report
}

// trimToFirstCycle returns text truncated to the end of its first repeating
// cycle. Returns the input unchanged when no cycle was found.
func trimToFirstCycle(text string, r repetitionReport) string {
	if !r.Looped() {
		return text
	}
	lines := strings.Split(text, "\n")
	idx, _ := significantLines(lines)
	if r.Period >= len(idx) {
		return text
	}
	// Cut immediately before the line that starts the second cycle so any
	// trailing blank lines or headings belonging to the first cycle survive.
	cut := idx[r.Period]
	return strings.TrimRight(strings.Join(lines[:cut], "\n"), " \t\n")
}

// significantLines returns the original line indices and the normalized text
// of lines long enough to be worth comparing.
func significantLines(lines []string) (idx []int, norm []string) {
	for i, line := range lines {
		n := normalizeForCompare(line)
		if len(n) < minSignificantLine {
			continue
		}
		idx = append(idx, i)
		norm = append(norm, n)
	}
	return idx, norm
}

// periodScore is the fraction of lines that match the line one period earlier
// in the sequence.
func periodScore(norm []string, period int) float64 {
	compared, matched := 0, 0
	for i := period; i < len(norm); i++ {
		compared++
		if norm[i] == norm[i-period] {
			matched++
		}
	}
	if compared == 0 {
		return 0
	}
	return float64(matched) / float64(compared)
}

// duplicateRatio is the fraction of lines that have appeared before. It
// catches partial degeneration that is not a clean cycle.
func duplicateRatio(norm []string) float64 {
	if len(norm) == 0 {
		return 0
	}
	seen := make(map[string]struct{}, len(norm))
	dupes := 0
	for _, n := range norm {
		if _, ok := seen[n]; ok {
			dupes++
			continue
		}
		seen[n] = struct{}{}
	}
	return float64(dupes) / float64(len(norm))
}

// normalizeForCompare collapses whitespace and case so that two cycles which
// differ only in spacing still compare equal.
func normalizeForCompare(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}
