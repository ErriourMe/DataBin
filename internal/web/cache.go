package web

import (
	"fmt"
	"net/http"
)

func (h *Handler) setRawCacheHeaders(w http.ResponseWriter, r *http.Request, id string) {
	maxAge, ok := h.pastes.CacheableRaw(r.Context(), id)
	if !ok {
		w.Header().Set("Cache-Control", "no-store")
		return
	}
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, immutable", maxAge))
}
