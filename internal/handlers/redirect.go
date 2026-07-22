package handlers

import (
	"net/http"
	"url-shortener/internal/store"
)
 
type RedirectHandler struct {
	Store *store.URLStore
}
 
func (h *RedirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
 
	entry, ok := h.Store.Get(code)
	if !ok {
		writeError(w, http.StatusNotFound, "Short URL not found or expired")
		return
	}
 
	referrer := r.Header.Get("Referer")
	userAgent := r.Header.Get("User-Agent")
 
	// This is the Go equivalent of FastAPI's BackgroundTasks: fire a goroutine
	// for analytics recording so the 302 response is never delayed by it.
	go entry.RecordClick(referrer, userAgent)
 
	http.Redirect(w, r, entry.OriginalURL, http.StatusFound)
}
