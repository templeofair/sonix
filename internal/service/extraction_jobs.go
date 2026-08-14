package service

import (
	"context"
	"sync"
)

type jobEntry struct {
	id     uint64
	cancel context.CancelFunc
}

// extractionJobs tracks in-flight RunExtraction cancel funcs per document.
type extractionJobs struct {
	mu     sync.Mutex
	nextID uint64
	byDoc  map[int64]jobEntry
	slots  *extractSlots
}

func newExtractionJobs() *extractionJobs {
	return newExtractionJobsWithSlots(1)
}

func newExtractionJobsWithSlots(n int) *extractionJobs {
	return &extractionJobs{byDoc: make(map[int64]jobEntry), slots: newExtractSlots(n)}
}

type extractSlots struct {
	ch chan struct{}
}

func newExtractSlots(n int) *extractSlots {
	if n < 1 {
		n = 1
	}
	return &extractSlots{ch: make(chan struct{}, n)}
}

func (s *extractSlots) try() bool {
	select {
	case s.ch <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *extractSlots) release() {
	<-s.ch
}

// track cancels any prior job for docID and registers cancel. Returns a job id
// used by untrack so a finished older goroutine cannot clear a newer job.
func (j *extractionJobs) track(docID int64, cancel context.CancelFunc) uint64 {
	j.mu.Lock()
	defer j.mu.Unlock()
	if prev, ok := j.byDoc[docID]; ok && prev.cancel != nil {
		prev.cancel()
	}
	j.nextID++
	id := j.nextID
	j.byDoc[docID] = jobEntry{id: id, cancel: cancel}
	return id
}

// untrack removes the job only if id is still the active registration.
func (j *extractionJobs) untrack(docID int64, id uint64) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if cur, ok := j.byDoc[docID]; ok && cur.id == id {
		delete(j.byDoc, docID)
	}
}

// cancel aborts the in-flight job for docID, if any.
func (j *extractionJobs) cancel(docID int64) {
	j.mu.Lock()
	entry, ok := j.byDoc[docID]
	if ok {
		delete(j.byDoc, docID)
	}
	j.mu.Unlock()
	if ok && entry.cancel != nil {
		entry.cancel()
	}
}
