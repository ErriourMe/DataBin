package web

import (
	"net/http"
	"strconv"
	"strings"
)

func burnAfterReadFromRequest(r *http.Request) bool {
	if v := strings.TrimSpace(r.URL.Query().Get("burn")); v == "1" || strings.EqualFold(v, "true") {
		return true
	}
	h := strings.TrimSpace(r.Header.Get("X-Databin-Burn"))
	if h == "1" || strings.EqualFold(h, "true") {
		return true
	}
	h = strings.TrimSpace(r.Header.Get("Burn-After-Read"))
	return h == "1" || strings.EqualFold(h, "true")
}

// burnShowsFromRequest returns user setting N (total opens = N+1). Default 1 when burn is enabled.
func burnShowsFromRequest(r *http.Request) (int, bool) {
	for _, src := range []struct {
		query  string
		header string
	}{
		{"burn_views", "X-Databin-Burn-Views"},
		{"burn_shows", "X-Databin-Burn-Shows"},
	} {
		raw := strings.TrimSpace(r.URL.Query().Get(src.query))
		if raw == "" {
			raw = strings.TrimSpace(r.Header.Get(src.header))
		}
		if raw == "" {
			continue
		}
		n, err := strconv.Atoi(raw)
		if err != nil {
			return 0, false
		}
		return n, true
	}
	return 1, true
}

func burnUniqueFromRequest(r *http.Request) bool {
	for _, raw := range []string{
		strings.TrimSpace(r.URL.Query().Get("burn_unique")),
		strings.TrimSpace(r.Header.Get("X-Databin-Burn-Unique")),
	} {
		if raw == "" {
			continue
		}
		if raw == "0" || strings.EqualFold(raw, "false") {
			return false
		}
		if raw == "1" || strings.EqualFold(raw, "true") {
			return true
		}
		return false
	}
	return true
}
