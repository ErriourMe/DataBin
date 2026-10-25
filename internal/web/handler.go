package web

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"databin/internal/config"
	"databin/internal/geoip"
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

func (h *Handler) Unlock(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := r.PathValue("id")
	info, exists, _ := h.pasteAccess(nil, r, id)
	if !exists {
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}
	if !info.PasswordProtected {
		http.Redirect(w, r, "/"+id, http.StatusSeeOther)
		return
	}
	if err := h.pastes.CheckPassword(r.Context(), id, r.FormValue("password")); err != nil {
		if errors.Is(err, paste.ErrWrongPassword) {
			h.renderPasswordGate(w, r, id, true)
			return
		}
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}
	setAccessCookie(w, h.cfg.AppSecret, id)
	lang := r.URL.Query().Get("lang")
	dest := "/" + id
	if lang != "" {
		dest += "?lang=" + lang
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	info, exists, authorized := h.pasteAccess(w, r, id)
	if !exists {
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}
	if !authorized {
		if isCLIClient(r) {
			if passwordFromRequest(r) != "" {
				http.Error(w, "incorrect password", http.StatusUnauthorized)
			} else {
				http.Error(w, "Password required (header X-Databin-Password)", http.StatusUnauthorized)
			}
			return
		}
		wrong := passwordFromRequest(r) != ""
		h.renderPasswordGate(w, r, id, wrong)
		return
	}
	read, ok := h.readPaste(w, r, id)
	if !ok {
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}
	h.recordPasteView(r, id)

	if isCLIClient(r) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if err := writeCLIHighlight(w, read.Content, r.URL.Query().Get("lang")); err != nil {
			_, _ = io.WriteString(w, read.Content)
		}
		return
	}

	data := h.viewBase(r)
	data["id"] = id
	data["content"] = read.Content
	data["burnViewsLeft"] = read.BurnViewsLeft
	data["burnLastView"] = read.BurnLastView
	data["burnUnique"] = info.BurnUnique
	data["lang"] = r.URL.Query().Get("lang")
	data["rawURL"] = "/raw/" + id
	data["downloadURL"] = "/download/" + id
	data["geoEnabled"] = geoip.Default() != nil
	data["ogTitle"] = id + " | " + h.cfg.AppName
	h.render(w, "paste_show.tmpl", data)
}

func (h *Handler) Raw(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !h.requirePasteAccess(w, r, id) {
		return
	}
	read, ok := h.readPaste(w, r, id)
	if !ok {
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}
	h.recordPasteView(r, id)
	h.setRawCacheHeaders(w, r, id)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(read.Content))
}

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !h.requirePasteAccess(w, r, id) {
		return
	}
	read, ok := h.readPaste(w, r, id)
	if !ok {
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}
	filename := id + ".txt"
	h.setRawCacheHeaders(w, r, id)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	h.recordPasteView(r, id)
	_, _ = w.Write([]byte(read.Content))
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

	burn := burnAfterReadFromRequest(r)
	opts := paste.StoreOptions{
		Burn:      burn,
		Password:  passwordForCreate(r),
		BurnShows: 1,
	}
	if burn {
		shows, ok := burnShowsFromRequest(r)
		if !ok {
			http.Error(w, paste.ErrInvalidBurnShows.Error(), http.StatusBadRequest)
			return
		}
		opts.BurnShows = shows
		opts.BurnUnique = burnUniqueFromRequest(r)
	}
	created, err := h.pastes.Store(r.Context(), body, opts)
	if err != nil {
		if errors.Is(err, paste.ErrInvalidBurnShows) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, paste.ErrPasswordTooLong) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, paste.ErrRedisMemoryPressure) {
			http.Error(w, err.Error(), http.StatusInsufficientStorage)
			return
		}
		if strings.Contains(err.Error(), "maximum size") {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Could not save paste", http.StatusInternalServerError)
		return
	}

	writeCreateResponse(w, created)
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

	if err := h.pastes.Update(r.Context(), id, body, tokenFromRequest(r)); err != nil {
		if errors.Is(err, paste.ErrNotFound) {
			http.Error(w, "Paste not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, paste.ErrManageTokenRequired) || errors.Is(err, paste.ErrInvalidManageToken) || errors.Is(err, paste.ErrManageTokenUnavailable) {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if errors.Is(err, paste.ErrEditingDisabled) || errors.Is(err, paste.ErrEditWindowClosed) {
			http.Error(w, err.Error(), http.StatusForbidden)
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
	if err := h.pastes.Delete(r.Context(), id, tokenFromRequest(r)); err != nil {
		if errors.Is(err, paste.ErrNotFound) {
			http.Error(w, "Paste not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, paste.ErrManageTokenRequired) || errors.Is(err, paste.ErrInvalidManageToken) || errors.Is(err, paste.ErrManageTokenUnavailable) {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if errors.Is(err, paste.ErrDeletionDisabled) || errors.Is(err, paste.ErrDeleteWindowClosed) {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Error(w, "Could not delete paste", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "OK")
}

func (h *Handler) requirePasteAccess(w http.ResponseWriter, r *http.Request, id string) bool {
	_, exists, authorized := h.pasteAccess(w, r, id)
	if !exists {
		http.Error(w, "Paste not found", http.StatusNotFound)
		return false
	}
	if !authorized {
		http.Error(w, "Password required", http.StatusUnauthorized)
		return false
	}
	return true
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
