package web

import (
	"fmt"
	"html/template"
	"path/filepath"
)

func LoadTemplates(dir string) (*template.Template, error) {
	pattern := filepath.Join(dir, "*.tmpl")
	tpl, err := template.ParseGlob(pattern)
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	return tpl, nil
}
