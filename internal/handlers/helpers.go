package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// checks that s is an absolute HTTP/HTTPS URL
func validateURL(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("url is required")
	}
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("url must start with http:// or https://")
	}
	if u.Host == "" {
		return "", fmt.Errorf("url must have a host")
	}
	return s, nil
}

func validateCustomCode(code string) error {
	if len(code) < 2 || len(code) > 64 {
		return fmt.Errorf("custom_code must be between 2 and 64 characters")
	}
	for _, c := range code {
		if !isAlphaNum(c) && c != '-' && c != '_' {
			return fmt.Errorf("custom_code may only contain letters, numbers, hyphens, and underscores")
		}
	}
	return nil
}

func isAlphaNum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}