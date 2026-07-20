package store

import (
	"sync"
	"testing"
	"time"
)

func makeEntry(code string, ttl time.Duration, maxClicks int) *URLEntry {
	var expiresAt *time.Time
	if ttl > 0 {
		t := time.Now().Add(ttl)
		expiresAt = &t
	}
	return NewEntry(code, "https://example.com/long/path", time.Now(), expiresAt, maxClicks)
}

func TestSaveAndGet(t *testing.T) {
	s := NewStore()
	s.Save(makeEntry("abc", 0, 100))

	entry, ok := s.Get("abc")
	if !ok || entry.Code != "abc" {
		t.Fatal("Expected to find saved entry")
	}
}

func TestGetMissingReturnsNotFound(t *testing.T) {
	s := NewStore()
	_, ok := s.Get("missing")
	if ok {
		t.Fatal("Expected not found for missing key")
	}
}

func TestExists(t *testing.T) {
	s := NewStore()
	s.Save(makeEntry("xyz", 0, 100))
	if !s.Exists("xyz") {
		t.Fatal("Expected Exists to return true")
	}
	if s.Exists("nope") {
		t.Fatal("Expected Exists to return false for missing key")
	}
}

func TestExpiredEntryReturnsNil(t *testing.T) {
	s := NewStore()
	s.Save(makeEntry("exp", 10*time.Millisecond, 100))
	time.Sleep(20 * time.Millisecond)
	_, ok := s.Get("exp")
	if ok {
		t.Fatal("Expected expired entry to return not found")
	}
}

func TestExpiredEntryEvictedOnGet(t *testing.T) {
	s := NewStore()
	s.Save(makeEntry("exp2", 10*time.Millisecond, 100))
	if s.Len() != 1 {
		t.Fatalf("Expected 1 entry, got %d", s.Len())
	}
	time.Sleep(20 * time.Millisecond)
	s.Get("exp2")
	if s.Len() != 0 {
		t.Fatal("Expected expired entry to be removed from map on Get")
	}
}

func TestEvictExpired(t *testing.T) {
	s := NewStore()
	s.Save(makeEntry("live", 0, 100))
	s.Save(makeEntry("dead", 10*time.Millisecond, 100))
	time.Sleep(20 * time.Millisecond)
	n := s.EvictExpired()
	if n != 1 {
		t.Fatalf("Expected 1 evicted, got %d", n)
	}
	if s.Len() != 1 {
		t.Fatalf("Expected 1 entry remaining, got %d", s.Len())
	}
}

func TestDelete(t *testing.T) {
	s := NewStore()
	s.Save(makeEntry("del", 0, 100))
	if !s.Delete("del") {
		t.Fatal("Expected Delete to return true")
	}
	if s.Delete("del") {
		t.Fatal("Expected second Delete to return false")
	}
	_, ok := s.Get("del")
	if ok {
		t.Fatal("Expected deleted entry to be gone")
	}
}

func TestRecordClickIncrementsCount(t *testing.T) {
	entry := makeEntry("x", 0, 100)
	entry.RecordClick("https://google.com", "Mozilla/5.0")
	entry.RecordClick("https://twitter.com", "")
	count, _, _ := entry.Stats()
	if count != 2 {
		t.Fatalf("Expected click count 2, got %d", count)
	}
}

func TestRecordClickTracksReferrers(t *testing.T) {
	entry := makeEntry("x", 0, 100)
	entry.RecordClick("https://google.com/search", "")
	entry.RecordClick("https://google.com/search", "")
	entry.RecordClick("https://twitter.com", "")
	_, refs, _ := entry.Stats()
	if refs["google.com"] != 2 {
		t.Fatalf("Expected google.com count 2, got %d", refs["google.com"])
	}
	if refs["twitter.com"] != 1 {
		t.Fatalf("Expected twitter.com count 1, got %d", refs["twitter.com"])
	}
}

func TestClickRingBufferDropsOldest(t *testing.T) {
	entry := makeEntry("x", 0, 3)
	for i := 0; i < 5; i++ {
		entry.RecordClick("", "")
	}
	count, _, clicks := entry.Stats()
	if count != 5 {
		t.Fatalf("Expected total count 5, got %d", count)
	}
	if len(clicks) != 3 {
		t.Fatalf("Expected 3 recent clicks, got %d", len(clicks))
	}
}

func TestConcurrentClickRecording(t *testing.T) {
	const goroutines = 50
	const clicksEach = 4 // 200 total

	entry := makeEntry("x", 0, 1000)
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < clicksEach; i++ {
				entry.RecordClick("https://example.com", "agent")
			}
		}()
	}
	wg.Wait()

	count, _, _ := entry.Stats()
	if count != goroutines*clicksEach {
		t.Fatalf("Expected %d clicks, got %d", goroutines*clicksEach, count)
	}
}