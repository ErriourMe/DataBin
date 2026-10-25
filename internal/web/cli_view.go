package web

import (
	"io"
	"net/http"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
)

func isCLIClient(r *http.Request) bool {
	ua := strings.ToLower(r.Header.Get("User-Agent"))
	if ua == "" {
		return false
	}
	return strings.Contains(ua, "curl") ||
		strings.Contains(ua, "wget") ||
		strings.Contains(ua, "httpie")
}

func writeCLIHighlight(w io.Writer, content, lang string) error {
	lexer := strings.TrimSpace(strings.ToLower(lang))
	if lexer == "" || lexer == "auto" || lexer == "plaintext" {
		lexer = ""
	}
	return quick.Highlight(w, content, lexer, "terminal", "monokai")
}
