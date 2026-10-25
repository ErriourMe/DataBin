package web

import (
	"errors"
	"net/http"
	"time"

	"databin/internal/paste"
)

const manageCookieName = "databin_manage"

func (h *Handler) manageAuthorized(w http.ResponseWriter, r *http.Request, id string) (exists bool, authorized bool) {
	_, exists = h.pastes.Info(r.Context(), id)
	if !exists {
		return false, false
	}
	token := tokenFromRequest(r)
	if token != "" {
		err := h.pastes.CheckManageToken(r.Context(), id, token)
		if err == nil {
			if w != nil {
				setManageCookie(w, h.cfg.AppSecret, id)
			}
			return true, true
		}
		if !errors.Is(err, paste.ErrInvalidManageToken) && !errors.Is(err, paste.ErrManageTokenRequired) {
			return true, false
		}
	}
	if manageCookieValid(r, h.cfg.AppSecret, id) {
		return true, true
	}
	return true, false
}

func setManageCookie(w http.ResponseWriter, secret, id string) {
	val := signManageToken(secret, id)
	http.SetCookie(w, &http.Cookie{
		Name:     manageCookieName,
		Value:    val,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((24 * time.Hour).Seconds()),
	})
}

func manageCookieValid(r *http.Request, secret, id string) bool {
	c, err := r.Cookie(manageCookieName)
	if err != nil || c.Value == "" {
		return false
	}
	return c.Value == signManageToken(secret, id)
}

func signManageToken(secret, id string) string {
	return signAccessToken(secret, "manage:"+id)
}
