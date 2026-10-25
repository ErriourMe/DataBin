package web

import (
	"net"
	"net/http"
	"strings"
)

func clientIP(r *http.Request) string {
	peer := remoteHost(r)
	if isTrustedProxy(peer) {
		if ip := forwardedClientIP(r); ip != "" {
			return ip
		}
	}
	return peer
}

func remoteHost(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}

func isTrustedProxy(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

func forwardedClientIP(r *http.Request) string {
	// Prefer X-Real-IP from nginx ($remote_addr); X-Forwarded-For may list an untrusted leftmost hop.
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	return forwardedForParam(r.Header.Get("Forwarded"))
}

func forwardedForParam(forwarded string) string {
	forwarded = strings.TrimSpace(forwarded)
	if forwarded == "" {
		return ""
	}
	for _, part := range strings.Split(forwarded, ";") {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(strings.ToLower(part), "for=") {
			continue
		}
		val := strings.TrimSpace(part[4:])
		val = strings.Trim(val, "\"")
		if strings.HasPrefix(val, "[") {
			end := strings.Index(val, "]")
			if end > 0 {
				return val[1:end]
			}
		}
		host, _, err := net.SplitHostPort(val)
		if err == nil {
			return host
		}
		return val
	}
	return ""
}
