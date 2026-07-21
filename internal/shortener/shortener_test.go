package shortener

import (
	"errors"
	"testing"
	"time"
	"url-shortener/internal/store"
)

func emptyStore() *store.URLStore {
	return store.NewStore()
}

func storeWith(codes ...string) *store.URLStore {
	s := store.NewStore()
	for _, code := range codes {
		s.Save(store.NewEntry(code, "https://example.com", time.Now(), nil, 100))
	}
	return s
}

func TestGenerateCodeLength(t *testing.T) {
	s := emptyStore()
	code, err := GenerateCode(s, 6, 5)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("Expected length 6, got %d", len(code))
	}
}

func TestGenerateCodeUnique(t *testing.T) {
	s := emptyStore()
	seen := make(map[string]bool)
	for i := 0; i < 20; i++ {
		code, err := GenerateCode(s, 8, 5)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if seen[code] {
			t.Fatalf("Duplicate code generated: %q", code)
		}
		seen[code] = true
	}
}

func TestGenerateCodeURLSafeCharacters(t *testing.T) {
	s := emptyStore()
	for i := 0; i < 50; i++ {
		code, _ := GenerateCode(s, 8, 5)
		for _, c := range code {
			if !isURLSafe(c) {
				t.Fatalf("Code %q contains non-URL-safe character %q", code, c)
			}
		}
	}
}

func isURLSafe(r rune) bool {
	return (r >= 'A' && r <= 'Z') ||
		(r >= 'a' && r <= 'z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '_'
}

func TestGenerateCodeErrorsAfterMaxRetries(t *testing.T) {
	s := storeWith("taken")
	_, err := ResolveCode(s, "taken", 6, 5)
	if err == nil {
		t.Fatal("Expected conflict error")
	}
	var conflict *ErrCodeConflict
	if !errors.As(err, &conflict) {
		t.Fatalf("Expected ErrCodeConflict, got %T", err)
	}
	if conflict.Code != "taken" {
		t.Fatalf("Expected code 'taken', got %q", conflict.Code)
	}
}
 
func TestResolveCodeUsesCustomWhenProvided(t *testing.T) {
	s := emptyStore()
	code, err := ResolveCode(s, "my-alias", 6, 5)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if code != "my-alias" {
		t.Fatalf("Expected 'my-alias', got %q", code)
	}
}
 
func TestResolveCodeGeneratesRandomWhenEmpty(t *testing.T) {
	s := emptyStore()
	code, err := ResolveCode(s, "", 6, 5)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("Expected length 6, got %d", len(code))
	}
}
