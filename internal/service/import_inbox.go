package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	inboxProcessedDir = "processed"
	inboxFailedDir    = "failed"
	defaultStableWait = 2 * time.Second
	defaultPollEvery  = 2 * time.Second
	// Match HTTP upload cap in internal/server/pages.go (50 MB).
	maxInboxImportBytes int64 = 50 << 20
)

// InboxImporter watches a folder and creates Sonix documents from dropped files.
// It has no knowledge of scanners or duplex — one finished file = one document.
type InboxImporter struct {
	Dir      string
	Docs     *DocumentService
	Extract  *ExtractionService
	Settings *SettingsService

	StableAfter time.Duration
	PollEvery   time.Duration

	// pending tracks size across polls for stability.
	pending map[string]int64
}

// NewInboxImporter builds an importer for dir. Dir is created if missing.
func NewInboxImporter(dir string, docs *DocumentService, extract *ExtractionService, settings *SettingsService) *InboxImporter {
	return &InboxImporter{
		Dir:         dir,
		Docs:        docs,
		Extract:     extract,
		Settings:    settings,
		StableAfter: defaultStableWait,
		PollEvery:   defaultPollEvery,
		pending:     make(map[string]int64),
	}
}

// EnsureDirs creates inbox, processed, and failed directories.
func (i *InboxImporter) EnsureDirs() error {
	for _, sub := range []string{"", inboxProcessedDir, inboxFailedDir} {
		p := i.Dir
		if sub != "" {
			p = filepath.Join(i.Dir, sub)
		}
		if err := os.MkdirAll(p, 0700); err != nil {
			return err
		}
	}
	return nil
}

// Run polls until ctx is cancelled.
func (i *InboxImporter) Run(ctx context.Context) {
	if i.PollEvery <= 0 {
		i.PollEvery = defaultPollEvery
	}
	if i.StableAfter <= 0 {
		i.StableAfter = defaultStableWait
	}
	log.Printf("inbox_import: watching %s (poll=%s stable=%s)", i.Dir, i.PollEvery, i.StableAfter)
	ticker := time.NewTicker(i.PollEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Printf("inbox_import: stopped")
			return
		case <-ticker.C:
			if i.Settings != nil && !i.Settings.ImportInboxEnabled(ctx) {
				continue
			}
			if err := i.ScanOnce(ctx); err != nil {
				log.Printf("inbox_import: scan error: %v", err)
			}
		}
	}
}

// ScanOnce processes stable files in the inbox root (not subdirs).
func (i *InboxImporter) ScanOnce(ctx context.Context) error {
	entries, err := os.ReadDir(i.Dir)
	if err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		ct, ok := contentTypeForInboxFile(name)
		if !ok {
			continue
		}
		path := filepath.Join(i.Dir, name)
		seen[path] = true
		fi, err := os.Lstat(path)
		if err != nil {
			continue
		}
		if fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
			log.Printf("inbox_import: skip non-regular file %s", name)
			continue
		}
		size := fi.Size()
		if size > maxInboxImportBytes {
			log.Printf("inbox_import: skip oversized %s (%d bytes)", name, size)
			i.recordResult(ctx, name, "file too large (max 50 MB)")
			_ = moveInboxFile(path, filepath.Join(i.Dir, inboxFailedDir, name))
			continue
		}
		prev, had := i.pending[path]
		if !had || prev != size {
			i.pending[path] = size
			continue
		}
		// Size unchanged for one full poll interval (~StableAfter when PollEvery≈StableAfter).
		delete(i.pending, path)
		if err := i.importFile(ctx, path, name, ct); err != nil {
			log.Printf("inbox_import: failed %s: %v", name, err)
			i.recordResult(ctx, name, PublicImportMessage(err.Error()))
			_ = moveInboxFile(path, filepath.Join(i.Dir, inboxFailedDir, name))
		}
	}
	for p := range i.pending {
		if !seen[p] {
			delete(i.pending, p)
		}
	}
	return nil
}

func (i *InboxImporter) importFile(ctx context.Context, path, name, contentType string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
		return fmt.Errorf("refusing non-regular file")
	}
	if fi.Size() > maxInboxImportBytes {
		return fmt.Errorf("file too large (max 50 MB)")
	}

	title := titleFromFilename(name)
	log.Printf("inbox_import: importing %s as %q", name, title)

	docID, err := i.Docs.Create(ctx, title)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}

	err = i.Docs.UploadPages(ctx, docID, []PageUpload{{
		ContentType: contentType,
		Open: func() (io.ReadCloser, error) {
			return os.Open(path)
		},
	}})
	if err != nil {
		_ = i.Docs.Delete(ctx, docID)
		return fmt.Errorf("upload: %w", err)
	}

	autoExtract := true
	if i.Settings != nil {
		autoExtract = i.Settings.ImportAutoExtract(ctx)
	}
	if autoExtract && i.Extract != nil {
		useOCR := true
		if i.Settings != nil {
			useOCR = i.Settings.ImportExtractUseOCR(ctx)
		}
		if err := i.Extract.Start(ctx, docID, useOCR); err != nil {
			log.Printf("inbox_import: extract start doc=%d: %v", docID, err)
		}
	}

	dest := filepath.Join(i.Dir, inboxProcessedDir, name)
	if err := moveInboxFile(path, dest); err != nil {
		log.Printf("inbox_import: move processed %s: %v", name, err)
	}
	i.recordResult(ctx, name, "")
	log.Printf("inbox_import: ok doc=%d file=%s", docID, name)
	return nil
}

func (i *InboxImporter) recordResult(ctx context.Context, name, errMsg string) {
	if i.Settings == nil {
		return
	}
	_ = i.Settings.RecordImportResult(ctx, name, errMsg)
}

func contentTypeForInboxFile(name string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".pdf":
		return "application/pdf", true
	case ".jpg", ".jpeg":
		return "image/jpeg", true
	case ".png":
		return "image/png", true
	case ".webp":
		return "image/webp", true
	default:
		return "", false
	}
}

func titleFromFilename(name string) string {
	base := filepath.Base(name)
	ext := filepath.Ext(base)
	t := strings.TrimSpace(strings.TrimSuffix(base, ext))
	if t == "" {
		return "Scanned letter"
	}
	return t
}

func moveInboxFile(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
		return err
	}
	if err := os.Rename(src, dest); err == nil {
		return nil
	}
	// Cross-device fallback
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Remove(src)
}
