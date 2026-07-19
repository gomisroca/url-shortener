package models

import "time"

// ShortenRequest is the JSON body for POST /shorten.
type ShortenRequest struct {
	URL        string  `json:"url"`
	CustomCode string  `json:"custom_code,omitempty"`
	TTLSeconds float64 `json:"ttl_seconds,omitempty"`
}

// ShortenResponse is the JSON body returned from POST /shorten.
type ShortenResponse struct {
	Code        string     `json:"code"`
	ShortURL    string     `json:"short_url"`
	OriginalURL string     `json:"original_url"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

// ClickEvent records one redirect hit.
type ClickEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Referrer  string    `json:"referrer,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
}

// StatsResponse is the JSON body returned from GET /stats/{code}.
type StatsResponse struct {
	Code         string            `json:"code"`
	OriginalURL  string            `json:"original_url"`
	CreatedAt    time.Time         `json:"created_at"`
	ExpiresAt    *time.Time        `json:"expires_at"`
	ClickCount   int64             `json:"click_count"`
	Referrers    map[string]int64  `json:"referrers"`
	RecentClicks []ClickEvent      `json:"recent_clicks"`
}