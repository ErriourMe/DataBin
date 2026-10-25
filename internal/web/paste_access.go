package web

import (
	"errors"
	"net/http"

	"databin/internal/paste"
)

func (h *Handler) pasteAccess(w http.ResponseWriter, r *http.Request, id string) (info paste.Info, exists bool, authorized bool) {
	info, exists = h.pastes.Info(r.Context(), id)
	if !exists {
		return info, false, false
	}
	if !info.PasswordProtected {
		return info, true, true
	}
	pwd := passwordFromRequest(r)
	if pwd != "" {
		err := h.pastes.CheckPassword(r.Context(), id, pwd)
		if err == nil {
			if w != nil {
				setAccessCookie(w, h.cfg.AppSecret, id)
			}
			return info, true, true
		}
		if !errors.Is(err, paste.ErrWrongPassword) {
			return info, true, false
		}
	}
	if accessCookieValid(r, h.cfg.AppSecret, id) {
		return info, true, true
	}
	return info, true, false
}

func (h *Handler) renderPasswordGate(w http.ResponseWriter, r *http.Request, id string, wrongPassword bool) {
	data := h.viewBase(r)
	data["id"] = id
	data["lang"] = r.URL.Query().Get("lang")
	data["wrongPassword"] = wrongPassword
	data["ogTitle"] = id + " | " + h.cfg.AppName
	h.render(w, "paste_password.tmpl", data)
}
