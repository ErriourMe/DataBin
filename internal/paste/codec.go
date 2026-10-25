package paste

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

const gzipPrefix = "dbgz\x00"

func encodeBody(plain string) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(gzipPrefix)
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write([]byte(plain)); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	out := buf.Bytes()
	if len(out) < len(plain) {
		return out, nil
	}
	return []byte(plain), nil
}

func decodeBody(raw []byte) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	if !bytes.HasPrefix(raw, []byte(gzipPrefix)) {
		return string(raw), nil
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw[len(gzipPrefix):]))
	if err != nil {
		return "", fmt.Errorf("gzip: %w", err)
	}
	defer zr.Close()
	out, err := io.ReadAll(zr)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
