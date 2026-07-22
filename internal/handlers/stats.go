package handlers

import (
	"net/http"
	"url-shortener/internal/models"
	"url-shortener/internal/store"
)

type StatsHandler struct {
	Store *store.URLStore
}

func (h *StatsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	entry, ok := h.Store.Get(code)
	if !ok {
		writeError(w, http.StatusNotFound, "Short URL not found or expired")
		return
	}

	count, refs, clicks := entry.Stats()

	writeJSON(w, http.StatusOK, models.StatsResponse{
		Code:         entry.Code,
		OriginalURL:  entry.OriginalURL,
		CreatedAt:    entry.CreatedAt,
		ExpiresAt:    entry.ExpiresAt,
		ClickCount:   count,
		Referrers:    refs,
		RecentClicks: clicks,
	})
}