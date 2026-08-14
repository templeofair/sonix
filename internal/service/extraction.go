package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/templeofair/sonix/internal/config"
	"github.com/templeofair/sonix/internal/ocr"
	"github.com/templeofair/sonix/internal/ollama"
	"github.com/templeofair/sonix/internal/repository"
)

var (
	// ErrExtractionNotFound is returned when status/reset targets a missing document.
	ErrExtractionNotFound = errors.New("extraction document not found")
	// ErrExtractionBusy is returned when EXTRACTION_MAX_CONCURRENT jobs are already running.
	ErrExtractionBusy = errors.New("extraction already running")
)

// rawResponseLimit caps what we store in extractions.raw_response. A few KB is
// plenty to see how a reply was malformed, and a degenerate model can produce
// megabytes we have no use for.
const rawResponseLimit = 4096

// ExtractionService orchestrates the OCR/vision + metadata pipeline.
type ExtractionService struct {
	extractions repository.ExtractionRepository
	settings    *SettingsService
	ocr         ocr.Provider
	cfg         *config.Config
	uploadsPath string
	jobs        *extractionJobs
}

// NewExtractionService wires extraction dependencies.
func NewExtractionService(
	extractions repository.ExtractionRepository,
	settings *SettingsService,
	ocrProvider ocr.Provider,
	cfg *config.Config,
	uploadsPath string,
) *ExtractionService {
	if ocrProvider == nil {
		ocrProvider = ocr.NewTesseract()
	}
	n := 1
	if cfg != nil && cfg.ExtractionMaxConcurrent > 0 {
		n = cfg.ExtractionMaxConcurrent
	}
	return &ExtractionService{
		extractions: extractions,
		settings:    settings,
		ocr:         ocrProvider,
		cfg:         cfg,
		uploadsPath: uploadsPath,
		jobs:        newExtractionJobsWithSlots(n),
	}
}

func (s *ExtractionService) jobTimeout() time.Duration {
	min := 60
	if s.cfg != nil && s.cfg.ExtractionJobTimeoutMin > 0 {
		min = s.cfg.ExtractionJobTimeoutMin
	}
	return time.Duration(min) * time.Minute
}

func (s *ExtractionService) useLegacyPipeline() bool {
	if s.cfg == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(s.cfg.ExtractionPipeline), "v1")
}

// Start marks the document processing and runs extraction in the background.
func (s *ExtractionService) Start(ctx context.Context, docID int64, useOCR bool) error {
	if !s.jobs.slots.try() {
		return ErrExtractionBusy
	}
	if err := s.extractions.SetDocumentProcessing(ctx, docID); err != nil {
		s.jobs.slots.release()
		return err
	}
	jobCtx, cancel := context.WithTimeout(context.Background(), s.jobTimeout())
	jobID := s.jobs.track(docID, cancel)
	go func() {
		defer cancel()
		defer s.jobs.untrack(docID, jobID)
		defer s.jobs.slots.release()
		s.RunExtraction(jobCtx, docID, useOCR)
	}()
	return nil
}

// Status returns the document status string.
func (s *ExtractionService) Status(ctx context.Context, docID int64) (string, error) {
	status, err := s.extractions.GetDocumentStatus(ctx, docID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", ErrExtractionNotFound
		}
		return "", err
	}
	return status, nil
}

// Reset aborts any in-flight Ollama work and moves processing/failed/partial back to pending.
func (s *ExtractionService) Reset(ctx context.Context, docID int64) error {
	s.jobs.cancel(docID)
	ok, err := s.extractions.ResetExtraction(ctx, docID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrExtractionNotFound
	}
	return nil
}

// ResetStuckExtractions marks every document stuck in "processing" as failed.
func (s *ExtractionService) ResetStuckExtractions(ctx context.Context) (int64, error) {
	return s.extractions.ResetStuckExtractions(ctx)
}

// ---------------------------------------------------------------------------
// Pipeline (moved from internal/server/extract.go)
// ---------------------------------------------------------------------------

// textExtractor produces the original (untranslated) text for one page.
type textExtractor interface {
	name() string
	engineID() string
	extract(ctx context.Context, fullPath string) (string, error)
}

type ocrTextExtractor struct {
	provider ocr.Provider
	lang     string
}

func (ocrTextExtractor) name() string { return "OCR" }

func (o ocrTextExtractor) engineID() string {
	if o.lang != "" {
		return "tesseract:" + o.lang
	}
	return o.provider.EngineID()
}

func (o ocrTextExtractor) extract(ctx context.Context, fullPath string) (string, error) {
	text, err := o.provider.ExtractText(ctx, fullPath, o.lang)
	if err != nil {
		return "", fmt.Errorf("OCR: %v", err)
	}
	return text, nil
}

type visionLLMExtractor struct {
	client *ollama.Client
}

func (visionLLMExtractor) name() string { return "LLM vision" }

func (v visionLLMExtractor) engineID() string {
	return "vision:" + v.client.PageProfileName()
}

func (v visionLLMExtractor) extract(ctx context.Context, fullPath string) (string, error) {
	tPrep := time.Now()
	b64, err := ollama.ImageToBase64ForVision(fullPath)
	prepMs := time.Since(tPrep).Milliseconds()
	if err != nil {
		return "", fmt.Errorf("read image: %v", err)
	}
	tHTTP := time.Now()
	text, _, err := v.client.ExtractPage(ctx, b64)
	httpMs := time.Since(tHTTP).Milliseconds()
	log.Printf("extraction: vision_timing prep_ms=%d http_ms=%d b64_chars=%d path=%s",
		prepMs, httpMs, len(b64), filepath.Base(fullPath))
	if err != nil {
		return "", fmt.Errorf("LLM vision: %v", err)
	}
	return strings.TrimSpace(text), nil
}

func (s *ExtractionService) pickExtractor(ctx context.Context, useOCR bool) textExtractor {
	if useOCR {
		lang := "deu+eng"
		if s.cfg != nil && strings.TrimSpace(s.cfg.OCRLang) != "" {
			lang = strings.TrimSpace(s.cfg.OCRLang)
		}
		// Prefer the provider's resolved language (startup may have fallen back).
		if t, ok := s.ocr.(*ocr.Tesseract); ok && strings.TrimSpace(t.Lang) != "" {
			lang = t.Lang
		}
		return ocrTextExtractor{provider: s.ocr, lang: lang}
	}
	return visionLLMExtractor{client: s.buildOllamaClient(ctx)}
}

type pipelineStrategy string

const (
	pipelineStrategyTwoPhaseOCR    pipelineStrategy = "two_phase_ocr"
	pipelineStrategyTwoPhaseVision pipelineStrategy = "two_phase_vision"
)

type pipelinePlan struct {
	strategy    pipelineStrategy
	extractor   textExtractor
	visionModel string
	textModel   string
}

func (s *ExtractionService) pickPipeline(ctx context.Context, docID int64, useOCR bool, pageCount int) pipelinePlan {
	extractor := s.pickExtractor(ctx, useOCR)
	metaClient := s.buildOllamaClient(ctx)
	textModel := ""
	if metaClient != nil {
		textModel = metaClient.TextModel
	}

	var strat pipelineStrategy
	var visionModel string
	switch ext := extractor.(type) {
	case ocrTextExtractor:
		strat = pipelineStrategyTwoPhaseOCR
		_ = ext
	case visionLLMExtractor:
		if ext.client != nil {
			visionModel = ext.client.VisionModel
		}
		strat = pipelineStrategyTwoPhaseVision
	default:
		strat = pipelineStrategyTwoPhaseVision
	}

	log.Printf("extraction: doc_id=%d pipeline=%s pages=%d engine_id=%s vision_model=%q text_model=%q",
		docID, strat, pageCount, extractor.engineID(), visionModel, textModel)

	return pipelinePlan{
		strategy:    strat,
		extractor:   extractor,
		visionModel: visionModel,
		textModel:   textModel,
	}
}

func (s *ExtractionService) buildOllamaClient(ctx context.Context) *ollama.Client {
	baseURL := ""
	if s.settings != nil {
		baseURL = s.settings.EffectiveBaseURL(ctx)
	}
	if baseURL == "" && s.cfg != nil {
		baseURL = s.cfg.OllamaBaseURL
	}
	if err := ValidateOllamaURL(baseURL); err != nil {
		log.Printf("extraction: refusing Ollama URL: %v", err)
		return nil
	}
	vision := ""
	text := ""
	if s.settings != nil {
		vision = s.settings.EffectiveVisionModel(ctx)
		text = s.settings.EffectiveTextModel(ctx)
	}
	if vision == "" {
		if s.cfg != nil {
			vision = s.cfg.OllamaVision
		}
		if vision == "" {
			vision = "gemma3:latest"
		}
	}
	if text == "" {
		if s.cfg != nil {
			text = s.cfg.OllamaText
		}
		if text == "" {
			text = vision
		}
	}
	return ollama.NewClient(baseURL, vision, text)
}

// RunExtraction is the background job started from Start.
func (s *ExtractionService) RunExtraction(ctx context.Context, docID int64, useOCR bool) {
	pages, err := s.extractions.LoadPages(ctx, docID)
	if err != nil {
		s.fail(ctx, docID, err.Error(), nil)
		return
	}
	if len(pages) == 0 {
		log.Printf("extraction: doc_id=%d failed: no pages", docID)
		s.fail(ctx, docID, "no pages", nil)
		return
	}

	plan := s.pickPipeline(ctx, docID, useOCR, len(pages))
	if v, ok := plan.extractor.(visionLLMExtractor); ok && v.client == nil {
		s.fail(ctx, docID, "Ollama URL is not allowed", nil)
		return
	}
	timer := newExtractionTimer(docID, len(pages), useOCR, plan.strategy)
	timer.logPipelineStart()

	extractor := plan.extractor
	engineID := extractor.engineID()
	pageParts, fullTextOriginal, err := s.extractOriginalTextSequential(ctx, docID, pages, extractor, timer)
	if err != nil {
		s.fail(ctx, docID, err.Error(), timer)
		return
	}

	if !s.extractions.IsProcessing(ctx, docID) {
		log.Printf("extraction: doc_id=%d skipped (no longer processing, may have been cancelled)", docID)
		s.clearProgress(ctx, docID)
		timer.logPipelineTotal("cancelled")
		return
	}
	if err := s.extractions.SaveOriginalText(ctx, docID, fullTextOriginal, engineID); err != nil {
		log.Printf("extraction: doc_id=%d save original text failed: %v", docID, err)
		s.fail(ctx, docID, "save original text: "+err.Error(), timer)
		return
	}
	log.Printf("extraction: doc_id=%d original text saved (len=%d pages=%d), starting metadata", docID, len(fullTextOriginal), len(pageParts))

	tMeta := time.Now()
	client := s.buildOllamaClient(ctx)
	if client == nil {
		s.partial(ctx, docID, "Ollama URL is not allowed", timer)
		return
	}
	var summary, fullTextEnglish, documentDate string
	var metaErr error
	if s.useLegacyPipeline() {
		log.Printf("extraction: doc_id=%d text pipeline=v1 (legacy ExtractMetadata)", docID)
		summary, fullTextEnglish, documentDate, metaErr = s.generateMetadata(ctx, pageParts, fullTextOriginal)
	} else {
		summary, fullTextEnglish, documentDate, metaErr = s.runTextPipeline(ctx, docID, client, pageParts, fullTextOriginal, timer)
	}
	if metaErr != nil {
		log.Printf("extraction: doc_id=%d LLM metadata failed: %v", docID, metaErr)
		if raw := ollama.RawResponseFor(metaErr, rawResponseLimit); raw != "" {
			if saveErr := s.extractions.SaveRawResponse(ctx, docID, raw); saveErr != nil {
				log.Printf("extraction: doc_id=%d save raw_response failed: %v", docID, saveErr)
			}
		}
		if !s.extractions.IsProcessing(ctx, docID) {
			log.Printf("extraction: doc_id=%d skipped after metadata error (no longer processing)", docID)
			s.clearProgress(ctx, docID)
			timer.logPipelineTotal("cancelled")
			return
		}
		s.extractions.RefreshFTS(ctx, docID, "", fullTextOriginal, "")
		s.partial(ctx, docID, "LLM metadata: "+metaErr.Error(), timer)
		return
	}
	timer.logPhase("metadata", time.Since(tMeta).Milliseconds())
	promptVersion := ollama.MetadataPromptVersion
	fullTextEnglish = s.postProcessTranslation(ctx, docID, client, fullTextOriginal, fullTextEnglish, timer)
	fullTextEnglish, translatePartial := ResolveEnglishTranslation(fullTextOriginal, fullTextEnglish)

	if !s.extractions.IsProcessing(ctx, docID) {
		log.Printf("extraction: doc_id=%d skipped (no longer processing)", docID)
		s.clearProgress(ctx, docID)
		timer.logPipelineTotal("cancelled")
		return
	}
	if err := s.extractions.SaveMetadata(ctx, docID, summary, fullTextEnglish, documentDate, promptVersion); err != nil {
		log.Printf("extraction: doc_id=%d save LLM result failed: %v", docID, err)
		s.fail(ctx, docID, "save extraction: "+err.Error(), timer)
		return
	}
	log.Printf("extraction: doc_id=%d metadata saved (prompt_version=%q document_date=%q)", docID, promptVersion, documentDate)

	s.extractions.RefreshFTS(ctx, docID, summary, fullTextOriginal, fullTextEnglish)
	if err := s.extractions.SaveWallMs(ctx, docID, timer.wallMs()); err != nil {
		log.Printf("extraction: doc_id=%d save extraction_wall_ms failed: %v", docID, err)
	}
	if translatePartial {
		log.Printf("extraction: doc_id=%d translation unusable; marking partial (original kept)", docID)
		s.partial(ctx, docID, "translation failed: no usable English text", timer)
		return
	}
	if err := s.extractions.MarkReady(ctx, docID); err != nil {
		log.Printf("extraction: doc_id=%d mark ready failed: %v", docID, err)
	}
	timer.logPipelineTotal("success")
	log.Printf("extraction: doc_id=%d success", docID)
}

func (s *ExtractionService) extractOriginalTextSequential(
	ctx context.Context,
	docID int64,
	pages []repository.PageRef,
	extractor textExtractor,
	timer *extractionTimer,
) ([]string, string, error) {
	total := len(pages)
	s.setProgress(ctx, docID, 0, total)
	parts := make([]string, 0, total)
	for i, p := range pages {
		if !s.extractions.IsProcessing(ctx, docID) {
			return nil, "", fmt.Errorf("extraction cancelled")
		}
		fullPath := filepath.Join(s.uploadsPath, p.StoragePath)
		text, err := s.extractPageWithRetry(ctx, docID, p.Index, extractor, fullPath, timer)
		if err != nil {
			log.Printf("extraction: doc_id=%d page_index=%d %s failed after retries: %v", docID, p.Index, extractor.name(), err)
			return nil, "", err
		}
		parts = append(parts, text)
		s.setProgress(ctx, docID, i+1, total)
	}
	return parts, strings.TrimSpace(strings.Join(parts, "\n\n")), nil
}

func (s *ExtractionService) extractPageWithRetry(
	ctx context.Context,
	docID int64,
	pageIndex int,
	extractor textExtractor,
	fullPath string,
	timer *extractionTimer,
) (string, error) {
	const maxAttempts = 3
	var lastErr error
	pageStart := time.Now()
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if !s.extractions.IsProcessing(ctx, docID) {
			return "", fmt.Errorf("extraction cancelled")
		}
		text, err := extractor.extract(ctx, fullPath)
		if err == nil {
			if timer != nil {
				timer.logPageStep(pageIndex, pageStepKind(extractor), time.Since(pageStart).Milliseconds())
			}
			return text, nil
		}
		lastErr = err
		log.Printf("extraction: doc_id=%d page_index=%d %s attempt %d/%d: %v", docID, pageIndex, extractor.name(), attempt, maxAttempts, err)
		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(attempt*250) * time.Millisecond):
			}
		}
	}
	return "", lastErr
}

func (s *ExtractionService) setProgress(ctx context.Context, docID int64, done, total int) {
	if err := s.extractions.SetProgress(ctx, docID, done, total); err != nil {
		log.Printf("extraction: doc_id=%d set progress failed: %v", docID, err)
	}
}

func (s *ExtractionService) clearProgress(ctx context.Context, docID int64) {
	if err := s.extractions.ClearProgress(ctx, docID); err != nil {
		log.Printf("extraction: doc_id=%d clear progress failed: %v", docID, err)
	}
}

func (s *ExtractionService) runTextPipeline(
	ctx context.Context,
	docID int64,
	client *ollama.Client,
	pageParts []string,
	fullTextOriginal string,
	timer *extractionTimer,
) (summary, fullTextEnglish, documentDate string, err error) {
	t0 := time.Now()
	pageEng, fullTextEnglish, err := client.TranslatePages(ctx, pageParts, fullTextOriginal, func(done, total int) {
		s.setProgress(ctx, docID, done, total)
	})
	if err != nil {
		return "", "", "", err
	}
	if timer != nil {
		timer.logPhase("translate", time.Since(t0).Milliseconds())
	}
	if !s.extractions.IsProcessing(ctx, docID) {
		return "", "", "", fmt.Errorf("extraction cancelled")
	}

	t1 := time.Now()
	// Prefer summarizing the English pages. If translation produced nothing
	// usable, summarize from the original pages so partial jobs still get an
	// English synopsis (many text models summarize non-English sources well
	// even when full-document translate prompts fail).
	summaryPages := pageEng
	if strings.TrimSpace(fullTextEnglish) == "" {
		log.Printf("extraction: doc_id=%d summary from original text (translation empty)", docID)
		summaryPages = pageParts
	}
	summary, sumErr := client.SummarizeDocument(ctx, summaryPages)
	if timer != nil {
		timer.logPhase("summary", time.Since(t1).Milliseconds())
	}
	if sumErr != nil {
		log.Printf("extraction: doc_id=%d summary failed: %v", docID, sumErr)
	}

	t2 := time.Now()
	documentDate, metaErr := client.ExtractDocumentDate(ctx, pageParts[0])
	if timer != nil {
		timer.logPhase("document_date", time.Since(t2).Milliseconds())
	}
	if metaErr != nil {
		log.Printf("extraction: doc_id=%d document_date failed: %v", docID, metaErr)
	}
	return summary, fullTextEnglish, documentDate, nil
}

func (s *ExtractionService) generateMetadata(ctx context.Context, pageParts []string, fullTextOriginal string) (summary, fullTextEnglish, documentDate string, err error) {
	client := s.buildOllamaClient(ctx)
	if client == nil {
		return "", "", "", fmt.Errorf("Ollama URL is not allowed")
	}
	return client.ExtractMetadata(ctx, pageParts, fullTextOriginal)
}

// ResolveEnglishTranslation decides stored english text after LLM translate.
// Non-English sources: empty or still-German output → ("", true) for MarkPartial.
// English sources: empty falls back to original (verbatim skip); never mark partial.
func ResolveEnglishTranslation(orig, eng string) (english string, markPartial bool) {
	eng = strings.TrimSpace(eng)
	if !ollama.LikelyNonEnglish(orig) {
		if eng == "" {
			return strings.TrimSpace(orig), false
		}
		return eng, false
	}
	if eng == "" || ollama.ShouldRetryTranslation(orig, eng) {
		return "", true
	}
	return eng, false
}

func (s *ExtractionService) postProcessTranslation(
	ctx context.Context,
	docID int64,
	client *ollama.Client,
	fullTextOriginal, fullTextEnglish string,
	timer *extractionTimer,
) string {
	if !ollama.ShouldOuterTranslateRetry(fullTextOriginal, fullTextEnglish) {
		return fullTextEnglish
	}
	if !s.extractions.IsProcessing(ctx, docID) {
		return fullTextEnglish
	}
	log.Printf("extraction: doc_id=%d translate-only retry (echo after page translate)", docID)
	t0 := time.Now()
	eng, err := client.TranslateFullTextEnglish(ctx, fullTextOriginal)
	if timer != nil {
		timer.logPhase("translate_only_retry", time.Since(t0).Milliseconds())
	}
	if err != nil {
		log.Printf("extraction: doc_id=%d translate-only failed: %v", docID, err)
		return fullTextEnglish
	}
	eng = strings.TrimSpace(eng)
	if eng == "" {
		log.Printf("extraction: doc_id=%d translate-only empty; keeping prior translation", docID)
		return fullTextEnglish
	}
	if ollama.TranslationEchoesOriginal(fullTextOriginal, eng) {
		log.Printf("extraction: doc_id=%d translate-only still echoes original; keeping prior", docID)
		return fullTextEnglish
	}
	log.Printf("extraction: doc_id=%d translate-only ok (len=%d)", docID, len(eng))
	return eng
}

func (s *ExtractionService) fail(ctx context.Context, docID int64, reason string, timer *extractionTimer) {
	if timer != nil {
		timer.logPipelineTotal("failed")
	}
	log.Printf("extraction: doc_id=%d failed: %s", docID, reason)
	_ = s.extractions.MarkFailed(ctx, docID, PublicExtractionMessage(reason))
}

func (s *ExtractionService) partial(ctx context.Context, docID int64, reason string, timer *extractionTimer) {
	if timer != nil {
		timer.logPipelineTotal("partial")
	}
	log.Printf("extraction: doc_id=%d partial: %s", docID, reason)
	_ = s.extractions.MarkPartial(ctx, docID, PublicExtractionMessage(reason))
}

// SaveOriginalText persists original text + engine_id (exported for tests).
func (s *ExtractionService) SaveOriginalText(ctx context.Context, docID int64, original, engineID string) error {
	return s.extractions.SaveOriginalText(ctx, docID, original, engineID)
}

// SaveMetadata persists LLM metadata + prompt_version (exported for tests).
// Clears any legacy auto-category on write (manual tags only).
func (s *ExtractionService) SaveMetadata(ctx context.Context, docID int64, summary, fullTextEnglish, documentDate, promptVersion string) error {
	return s.extractions.SaveMetadata(ctx, docID, summary, fullTextEnglish, documentDate, promptVersion)
}

// SaveExtractionWallMs persists wall-clock ms (exported for tests).
func (s *ExtractionService) SaveExtractionWallMs(ctx context.Context, docID int64, wallMs int64) error {
	return s.extractions.SaveWallMs(ctx, docID, wallMs)
}

// PickPipeline selects the page text extractor (exported for tests).
func (s *ExtractionService) PickPipeline(ctx context.Context, docID int64, useOCR bool, pageCount int) (strategy, engineID, visionModel, textModel string, isOCR bool) {
	p := s.pickPipeline(ctx, docID, useOCR, pageCount)
	_, isOCR = p.extractor.(ocrTextExtractor)
	return string(p.strategy), p.extractor.engineID(), p.visionModel, p.textModel, isOCR
}
