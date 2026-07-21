package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
	"url-shortener/internal/models"
	"url-shortener/internal/shortener"
	"url-shortener/internal/store"
)

type ShortenHandler struct {
	Store 				*store.URLStore
	BaseURL 			string
	CodeLength 			int
	MaxCollisionRetries int
	DefaultTTL 			time.Duration
	MaxClicksPerURL 	int
}

func (h *ShortenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Only POST is supported")
		return
	}

	var req models.ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	validURL, err := validateURL(req.URL)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	if req.CustomCode != "" {
		if err := validateCustomCode(req.CustomCode); err != nil {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
	}

	code, err := shortener.ResolveCode(h.Store, req.CustomCode, h.CodeLength, h.MaxCollisionRetries)
	if err != nil {
		var conflict *shortener.ErrCodeConflict
		if errors.As(err, &conflict) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	var expiresAt *time.Time
	ttl := h.DefaultTTL
	if req.TTLSeconds > 0 {
		ttl = time.Duration(req.TTLSeconds * float64(time.Second))
	}
	if ttl > 0 {
		t := time.Now().UTC().Add(ttl)
		expiresAt = &t
	}
 
	entry := store.NewEntry(code, validURL, time.Now().UTC(), expiresAt, h.MaxClicksPerURL)
	h.Store.Save(entry)
 
	writeJSON(w, http.StatusOK, models.ShortenResponse{
		Code:        code,
		ShortURL:    fmt.Sprintf("%s/r/%s", h.BaseURL, code),
		OriginalURL: validURL,
		ExpiresAt:   expiresAt,
	})

}

type DeleteHandler struct {
	Store *store.URLStore
}
 
func (h *DeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "Only DELETE is supported")
		return
	}
	code := r.PathValue("code")
	if !h.Store.Delete(code) {
		writeError(w, http.StatusNotFound, "Short URL not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": code})
}
