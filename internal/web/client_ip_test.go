package web

import (
	"net/http/httptest"
	"testing"
)

func TestClientIPBehindProxy(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "172.17.0.1:54321"
	r.Header.Set("X-Real-IP", "203.0.113.50")
	if got := clientIP(r); got != "203.0.113.50" {
		t.Fatalf("got %q want 203.0.113.50", got)
	}
}

func TestClientIPPrefersRealIPOverXFF(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "172.17.0.1:54321"
	r.Header.Set("X-Forwarded-For", "172.17.0.1, 203.0.113.50")
	r.Header.Set("X-Real-IP", "203.0.113.50")
	if got := clientIP(r); got != "203.0.113.50" {
		t.Fatalf("got %q want 203.0.113.50", got)
	}
}

func TestClientIPDirect(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "203.0.113.50:12345"
	r.Header.Set("X-Real-IP", "1.2.3.4")
	if got := clientIP(r); got != "203.0.113.50" {
		t.Fatalf("got %q want 203.0.113.50 (ignore spoofed header)", got)
	}
}
