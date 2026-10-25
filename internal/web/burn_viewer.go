package web

import (
	"net/http"
	"strings"

	"databin/internal/paste"
)

const burnViewCookiePrefix = "databin_bv_"

// burnViewerKey returns a stable id per browser (cookie) or per IP before the cookie is stored.
func burnViewerKey(w http.ResponseWriter, r *http.Request, pasteID string) string {
	name := burnViewCookiePrefix + pasteID
	if c, err := r.Cookie(name); err == nil {
		v := strings.TrimSpace(c.Value)
		if v != "" {
			return v
		}
	}
	v := "ip:" + clientIP(r)
	if w != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    v,
			Path:     "/" + pasteID,
			MaxAge:   86400 * 365,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}
	return v
}

func (h *Handler) readPaste(w http.ResponseWriter, r *http.Request, id string) (out paste.ReadOutcome, ok bool) {
	info, exists := h.pastes.Info(r.Context(), id)
	if !exists {
		return out, false
	}
	viewer := ""
	if info.Burn {
		viewer = burnViewerKey(w, r, id)
	}
	return h.pastes.Read(r.Context(), id, viewer)
}
