package web

import (
	"io"
	"net/http"
	"strings"
)

func readBody(r *http.Request, maxBytes int) (string, error) {
	var data []byte
	if r.Body != nil {
		limited := io.LimitReader(r.Body, int64(maxBytes)+1)
		var err error
		data, err = io.ReadAll(limited)
		if err != nil {
			return "", err
		}
		if len(data) > maxBytes {
			return "", io.ErrShortBuffer
		}
		if len(data) > 0 {
			return string(data), nil
		}
	}

	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		if r.PostForm == nil {
			_ = r.ParseForm()
		}
		if body := bodyFromForm(r.PostForm); body != "" {
			if len(body) > maxBytes {
				return "", io.ErrShortBuffer
			}
			return body, nil
		}
	}

	return string(data), nil
}

func bodyFromForm(form map[string][]string) string {
	if len(form) == 0 {
		return ""
	}
	if len(form) == 1 {
		for key, values := range form {
			if len(values) == 0 || (len(values) == 1 && values[0] == "") {
				return key
			}
			return values[0]
		}
	}
	var parts []string
	for key, values := range form {
		if len(values) == 0 {
			parts = append(parts, key)
			continue
		}
		for _, v := range values {
			if v == "" {
				parts = append(parts, key)
			} else {
				parts = append(parts, v)
			}
		}
	}
	return strings.Join(parts, "\n")
}
