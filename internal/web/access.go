package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
	"time"
)

const accessCookieName = "databin_access"

func passwordFromRequest(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-Databin-Password")); v != "" {
		return v
	}
	if v := strings.TrimSpace(r.URL.Query().Get("password")); v != "" {
		return v
	}
	return strings.TrimSpace(r.FormValue("password"))
}

func passwordForCreate(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-Databin-Password"))
}

func tokenFromRequest(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-Databin-Token")); v != "" {
		return v
	}
	if v := strings.TrimSpace(r.URL.Query().Get("token")); v != "" {
		return v
	}
	return strings.TrimSpace(r.FormValue("token"))
}

func setAccessCookie(w http.ResponseWriter, secret, id string) {
	val := signAccessToken(secret, id)
	http.SetCookie(w, &http.Cookie{
		Name:     accessCookieName,
		Value:    val,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((24 * time.Hour).Seconds()),
	})
}

func accessCookieValid(r *http.Request, secret, id string) bool {
	c, err := r.Cookie(accessCookieName)
	if err != nil || c.Value == "" {
		return false
	}
	parts := strings.SplitN(c.Value, ".", 2)
	if len(parts) != 2 || parts[0] != id {
		return false
	}
	mac, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	expected := accessMAC(secret, id)
	return hmac.Equal(mac, expected)
}

func signAccessToken(secret, id string) string {
	mac := accessMAC(secret, id)
	return id + "." + base64.RawURLEncoding.EncodeToString(mac)
}

func accessMAC(secret, id string) []byte {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte("paste:"))
	h.Write([]byte(id))
	return h.Sum(nil)
}
