package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"databin/internal/geoip"
	"databin/internal/paste"
)

func (h *Handler) recordPasteView(r *http.Request, id string) {
	cc := geoip.CountryCode(clientIP(r))
	h.pastes.RecordView(r.Context(), id, cc)
}

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	exists, authorized := h.manageAuthorized(w, r, id)
	if !exists {
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}
	if !authorized {
		wrong := tokenFromRequest(r) != ""
		h.renderStatsGate(w, r, id, wrong)
		return
	}
	stats, err := h.pastes.GetStats(r.Context(), id)
	if err != nil {
		if errors.Is(err, paste.ErrNotFound) {
			http.Error(w, "Paste not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Could not load stats", http.StatusInternalServerError)
		return
	}
	if wantsJSON(r) {
		writeStatsJSON(w, stats)
		return
	}
	data := h.viewBase(r)
	data["id"] = id
	data["stats"] = stats
	data["dailyRows"] = dailyBarRows(stats.Daily)
	data["geoEnabled"] = geoip.Default() != nil
	data["ogTitle"] = id + " stats | " + h.cfg.AppName
	h.render(w, "paste_stats.tmpl", data)
}

func (h *Handler) StatsUnlock(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := r.PathValue("id")
	exists, _ := h.manageAuthorized(nil, r, id)
	if !exists {
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}
	if err := h.pastes.CheckManageToken(r.Context(), id, r.FormValue("token")); err != nil {
		if errors.Is(err, paste.ErrInvalidManageToken) || errors.Is(err, paste.ErrManageTokenRequired) {
			h.renderStatsGate(w, r, id, true)
			return
		}
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}
	setManageCookie(w, h.cfg.AppSecret, id)
	http.Redirect(w, r, "/"+id+"/stats", http.StatusSeeOther)
}

func (h *Handler) DocumentStats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	exists, authorized := h.manageAuthorized(w, r, id)
	w.Header().Set("Cache-Control", "no-store")
	if !exists {
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}
	if !authorized {
		http.Error(w, paste.ErrManageTokenRequired.Error(), http.StatusUnauthorized)
		return
	}
	stats, err := h.pastes.GetStats(r.Context(), id)
	if err != nil {
		if errors.Is(err, paste.ErrNotFound) {
			http.Error(w, "Paste not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Could not load stats", http.StatusInternalServerError)
		return
	}
	writeStatsJSON(w, stats)
}

func (h *Handler) renderStatsGate(w http.ResponseWriter, r *http.Request, id string, wrongToken bool) {
	data := h.viewBase(r)
	data["id"] = id
	data["wrongToken"] = wrongToken
	data["ogTitle"] = id + " stats | " + h.cfg.AppName
	h.render(w, "paste_stats_gate.tmpl", data)
}

func wantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

func writeStatsJSON(w http.ResponseWriter, stats paste.Stats) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(stats)
}

type statsDailyRow struct {
	Date     string
	Count    int64
	BarWidth int
}

func dailyBarRows(daily []paste.DailyView) []statsDailyRow {
	max := maxDailyCount(daily)
	rows := make([]statsDailyRow, len(daily))
	for i, d := range daily {
		w := 0
		if max > 0 {
			w = int(d.Count * 100 / max)
		}
		rows[i] = statsDailyRow{Date: d.Date, Count: d.Count, BarWidth: w}
	}
	return rows
}

func maxDailyCount(daily []paste.DailyView) int64 {
	var m int64
	for _, d := range daily {
		if d.Count > m {
			m = d.Count
		}
	}
	return m
}
