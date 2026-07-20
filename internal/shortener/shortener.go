package shortener

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"url-shortener/internal/store"
)

// Returned when a code is already in use.
type ErrCodeConflict struct {
	Code string
}

func (e *ErrCodeConflict) Error() string {
	return fmt.Sprintf("Code %q is already in use", e.Code)
}

// Returned when random code generation fails after maxRetries.
var ErrCodeGeneration = errors.New("Failed to generate a unique code after max retries")


// Generate a unique random code of codeLength URL-safe characters.
func GenerateCode(s *store.URLStore, codeLength, maxRetries int) (string, error) {
	for attempt := 0; attempt < maxRetries; attempt++ {
		rawBytes := make([]byte, codeLength)
		if _, err := rand.Read(rawBytes); err != nil {
			return "", fmt.Errorf("cryto/rand: %w", err)
		}

		candidate := base64.RawURLEncoding.EncodeToString(rawBytes)[:codeLength]
		if !s.Exists(candidate) {
			return candidate, nil
		}
	}
	return "", ErrCodeGeneration
}

// Return the custom code if provided, otherwise generate a random one.
func ResolveCode(s *store.URLStore, customCode string, codeLength, maxRetries int) (string, error) {
	if customCode != "" {
		if s.Exists(customCode) {
			return "", &ErrCodeConflict{Code: customCode}
		}
		return customCode, nil
	}
	return GenerateCode(s, codeLength, maxRetries)
}