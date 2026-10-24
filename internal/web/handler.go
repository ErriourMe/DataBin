package web

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"databin/internal/config"
	"databin/internal/paste"
	"databin/internal/version"
)

type Handler struct {
	cfg      config.Config
	pastes   *paste.Service
	versions *version.Service
	tpl      *template.Template
}

func NewHandler(cfg config.Config, pastes *paste.Service, versions *version.Service, tpl *template.Template) *Handler {
	return &Handler{
		cfg:      cfg,
		pastes:   pastes,
		versions: versions,
		tpl:      tpl,
	}
}

func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	data := h.viewBase(r)
	data["ogTitle"] = h.cfg.AppName
	h.render(w, "paste_new.tmpl", data)
}

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	content, ok := h.pastes.Get(r.Context(), id)
	if !ok {
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}

	data := h.viewBase(r)
	data["id"] = id
	data["content"] = content
	data["lang"] = r.URL.Query().Get("lang")
	data["rawURL"] = "/raw/" + id
	data["downloadURL"] = "/download/" + id
	data["ogTitle"] = id + " | " + h.cfg.AppName
	h.render(w, "paste_show.tmpl", data)
}

func (h *Handler) Raw(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	content, ok := h.pastes.Get(r.Context(), id)
	if !ok {
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(content))
}

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	content, ok := h.pastes.Get(r.Context(), id)
	if !ok {
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}
	filename := id + ".txt"
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	_, _ = w.Write([]byte(content))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	body, err := readBody(r, h.cfg.Paste.MaxBytes)
	if err != nil {
		http.Error(w, "Paste too large", http.StatusRequestEntityTooLarge)
		return
	}
	if isEmptyPaste(body) {
		http.Error(w, "Paste is empty", http.StatusBadRequest)
		return
	}

	id, err := h.pastes.Store(r.Context(), body)
	if err != nil {
		if strings.Contains(err.Error(), "maximum size") {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Could not save paste", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, id)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	body, err := readBody(r, h.cfg.Paste.MaxBytes)
	if err != nil {
		http.Error(w, "Paste too large", http.StatusRequestEntityTooLarge)
		return
	}
	if isEmptyPaste(body) {
		http.Error(w, "Paste is empty", http.StatusBadRequest)
		return
	}

	if err := h.pastes.Update(r.Context(), id, body); err != nil {
		if err.Error() == "paste not found" {
			http.Error(w, "Paste not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "maximum size") {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, id)
}

func (h *Handler) Destroy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !h.pastes.Delete(r.Context(), id) {
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "OK")
}

func (h *Handler) viewBase(r *http.Request) map[string]any {
	now := time.Now()
	release := h.versions.Current()
	base := strings.TrimRight(h.appURL(r), "/")
	data := map[string]any{
		"appName":         h.cfg.AppName,
		"appVersion":      release.Version,
		"appVersionURL":   release.URL,
		"footerCopyright": footerCopyright(now.Year()),
		"ogImageURL":      base + "/public/og.png",
	}
	if release.UpdateVersion != "" {
		data["updateVersion"] = release.UpdateVersion
		data["updateVersionURL"] = release.UpdateURL
	}
	return data
}

func (h *Handler) appURL(r *http.Request) string {
	if u := strings.TrimSpace(h.cfg.AppURL); u != "" {
		return strings.TrimRight(u, "/")
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	} else if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = strings.TrimSpace(strings.Split(proto, ",")[0])
	}
	return scheme + "://" + r.Host
}

func isEmptyPaste(content string) bool {
	return strings.TrimSpace(content) == ""
}

func footerCopyright(year int) string {
	const projectStart = 2026
	if year == projectStart {
		return "2026"
	}
	return fmt.Sprintf("%d-%d", projectStart, year)
}

func (h *Handler) render(w http.ResponseWriter, name string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}
