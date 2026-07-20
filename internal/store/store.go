// Holds shortened URLs and their click analytics in memory.

package store

import (
	"context"
	"net/url"
	"sync"
	"time"

	"url-shortener/internal/models"
)

// URLEntry holds one shortened URL and all its analytics state.
type URLEntry struct {
	Code        string
	OriginalURL string
	CreatedAt   time.Time
	ExpiresAt   *time.Time
	MaxClicks   int

	// Everything below is protected by mu.
	mu            sync.Mutex
	ClickCount    int64
	Clicks        []models.ClickEvent
	ReferrerCount map[string]int64
}

func NewEntry(code, originalURL string, createdAt time.Time, expiresAt *time.Time, maxClicks int) *URLEntry {
	return &URLEntry{
		Code:          code,
		OriginalURL:   originalURL,
		CreatedAt:     createdAt,
		ExpiresAt:     expiresAt,
		MaxClicks:     maxClicks,
		ReferrerCount: make(map[string]int64),
	}
}

func (e *URLEntry) IsExpired() bool {
	if e.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*e.ExpiresAt)
}

// RecordClick is safe to call from a goroutine after the response is sent.
// It increments the counter, appends the event, and trims the ring buffer.
func (e *URLEntry) RecordClick(referrer, userAgent string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.ClickCount++

	event := models.ClickEvent{
		Timestamp: time.Now().UTC(),
		Referrer:  referrer,
		UserAgent: userAgent,
	}
	e.Clicks = append(e.Clicks, event)

	// Ring buffer: keep only the most recent MaxClicks events.
	if len(e.Clicks) > e.MaxClicks {
		e.Clicks = e.Clicks[len(e.Clicks)-e.MaxClicks:]
	}

	if referrer != "" {
		domain := extractDomain(referrer)
		e.ReferrerCount[domain]++
	}
}

// Stats returns a snapshot of click analytics under the lock.
func (e *URLEntry) Stats() (int64, map[string]int64, []models.ClickEvent) {
	e.mu.Lock()
	defer e.mu.Unlock()

	refs := make(map[string]int64, len(e.ReferrerCount))
	for k, v := range e.ReferrerCount {
		refs[k] = v
	}

	// Return newest-first, same as the Python version.
	clicks := make([]models.ClickEvent, len(e.Clicks))
	for i, c := range e.Clicks {
		clicks[len(e.Clicks)-1-i] = c
	}

	return e.ClickCount, refs, clicks
}

func extractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return rawURL
	}
	return u.Host
}

// URLStore is the map of all live entries, protected by an RWMutex.
// Read-heavy operations (Get, Exists) take the read lock; writes take the full lock.
type URLStore struct {
	mu      sync.RWMutex
	entries map[string]*URLEntry
}

func NewStore() *URLStore {
	return &URLStore{entries: make(map[string]*URLEntry)}
}

func (s *URLStore) Save(entry *URLEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries[entry.Code] = entry
}

func (s *URLStore) Get(code string) (*URLEntry, bool) {
	s.mu.RLock()
	entry, ok := s.entries[code]
	s.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if entry.IsExpired() {
		s.mu.Lock()
		delete(s.entries, code)
		s.mu.Unlock()
		return nil, false
	}

	return entry, true
}

func (s *URLStore) Exists(code string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.entries[code]

	return ok
}

func (s *URLStore) Delete(code string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.entries[code]; !ok {
		return false
	}

	delete(s.entries, code)

	return true
}

func (s *URLStore) EvictExpired() int {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	n := 0
	for code, entry := range s.entries {
		if entry.ExpiresAt != nil && now.After(*entry.ExpiresAt) {
			delete(s.entries, code)
			n++
		}
	}

	return n
}

func (s *URLStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.entries)
}

// Background goroutine that periodically evicts expired entries. Stops cleanly when ctx is cancelled.
func (s *URLStore) StartCleanup(ctx context.Context, interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.EvictExpired()
			}
		}
	}()
}