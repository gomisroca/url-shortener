package handlers

import (
	"fmt"
	"net/http"

	"urlshortener/internal/store"
)

type HealthHandler struct {
	Store *store.URLStore
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"links":  fmt.Sprintf("%d", h.Store.Len()),
	})
}