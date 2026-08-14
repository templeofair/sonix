package service

import (
	"log"
	"strconv"
	"time"
)

// extractionTimer records wall-clock time for one RunExtraction job and emits
// grep-friendly lines with prefix "extraction_timing" (see docs/extraction.md).
type extractionTimer struct {
	docID    int64
	pages    int
	start    time.Time
	useOCR   bool
	pipeline string
}

func newExtractionTimer(docID int64, pageCount int, useOCR bool, strat pipelineStrategy) *extractionTimer {
	return &extractionTimer{
		docID:    docID,
		pages:    pageCount,
		start:    time.Now(),
		useOCR:   useOCR,
		pipeline: string(strat),
	}
}

func (t *extractionTimer) cumulativeMs() int64 {
	if t == nil {
		return 0
	}
	return time.Since(t.start).Milliseconds()
}

// wallMs returns elapsed ms since pipeline_start (same basis as pipeline_total duration_ms).
func (t *extractionTimer) wallMs() int64 {
	return t.cumulativeMs()
}

func (t *extractionTimer) logPipelineStart() {
	if t == nil {
		return
	}
	log.Printf("extraction_timing doc_id=%d event=pipeline_start pages=%d use_ocr=%s pipeline=%s cumulative_pipeline_ms=0",
		t.docID, t.pages, strconv.FormatBool(t.useOCR), t.pipeline)
}

func (t *extractionTimer) logPageStep(pageIndex int, kind string, durationMs int64) {
	if t == nil {
		return
	}
	log.Printf("extraction_timing doc_id=%d event=page_step page_index=%d kind=%s duration_ms=%d cumulative_pipeline_ms=%d",
		t.docID, pageIndex, kind, durationMs, t.cumulativeMs())
}

func (t *extractionTimer) logPhase(phase string, durationMs int64) {
	if t == nil {
		return
	}
	log.Printf("extraction_timing doc_id=%d event=phase phase=%s duration_ms=%d cumulative_pipeline_ms=%d",
		t.docID, phase, durationMs, t.cumulativeMs())
}

func (t *extractionTimer) logPipelineTotal(outcome string) {
	if t == nil {
		return
	}
	ms := time.Since(t.start).Milliseconds()
	log.Printf("extraction_timing doc_id=%d event=pipeline_total outcome=%s pages=%d duration_ms=%d cumulative_pipeline_ms=%d",
		t.docID, outcome, t.pages, ms, ms)
}

func pageStepKind(extractor textExtractor) string {
	switch extractor.(type) {
	case ocrTextExtractor:
		return "ocr"
	case visionLLMExtractor:
		return "vision_per_page"
	default:
		return "page_text"
	}
}
