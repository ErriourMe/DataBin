package controllers

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	frameworkhttp "github.com/goravel/framework/contracts/http"

	"goravel/app/facades"
	"goravel/app/services"
)

type PasteController struct {
	pastes *services.PasteService
}

func NewPasteController() *PasteController {
	return &PasteController{
		pastes: services.NewPasteService(),
	}
}

func (c *PasteController) New(ctx frameworkhttp.Context) frameworkhttp.Response {
	return ctx.Response().View().Make("paste_new.tmpl", viewBase())
}

func viewBase() map[string]any {
	now := time.Now()
	return map[string]any{
		"appName":         facades.Config().GetString("app.name", "DataBin"),
		"footerCopyright": footerCopyright(now.Year()),
	}
}

func footerCopyright(year int) string {
	const projectStart = 2026
	if year == projectStart {
		return "2026"
	}
	return fmt.Sprintf("%d-%d", projectStart, year)
}

func (c *PasteController) Show(ctx frameworkhttp.Context) frameworkhttp.Response {
	id := ctx.Request().Route("id")
	content, ok := c.pastes.Get(id)
	if !ok {
		return ctx.Response().Status(http.StatusNotFound).String("Paste not found")
	}

	lang := ctx.Request().Query("lang", "")
	data := viewBase()
	data["id"] = id
	data["content"] = content
	data["lang"] = lang
	data["rawURL"] = "/raw/" + id
	data["downloadURL"] = "/download/" + id
	return ctx.Response().View().Make("paste_show.tmpl", data)
}

func (c *PasteController) Raw(ctx frameworkhttp.Context) frameworkhttp.Response {
	id := ctx.Request().Route("id")
	content, ok := c.pastes.Get(id)
	if !ok {
		return ctx.Response().Status(http.StatusNotFound).String("Paste not found")
	}
	return ctx.Response().Data(http.StatusOK, "text/plain; charset=utf-8", []byte(content))
}

func (c *PasteController) Download(ctx frameworkhttp.Context) frameworkhttp.Response {
	id := ctx.Request().Route("id")
	content, ok := c.pastes.Get(id)
	if !ok {
		return ctx.Response().Status(http.StatusNotFound).String("Paste not found")
	}
	filename := id + ".txt"
	return ctx.Response().
		Header("Content-Type", "text/plain; charset=utf-8").
		Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename)).
		Data(http.StatusOK, "text/plain; charset=utf-8", []byte(content))
}

func (c *PasteController) Create(ctx frameworkhttp.Context) frameworkhttp.Response {
	body, err := readBody(ctx, facades.Config().GetInt("paste.max_bytes", 2*1024*1024))
	if err != nil {
		return ctx.Response().Status(http.StatusRequestEntityTooLarge).String("Paste too large")
	}

	id, err := c.pastes.Store(body)
	if err != nil {
		if strings.Contains(err.Error(), "maximum size") {
			return ctx.Response().Status(http.StatusRequestEntityTooLarge).String(err.Error())
		}
		return ctx.Response().Status(http.StatusInternalServerError).String("Could not save paste")
	}

	return ctx.Response().Header("Content-Type", "text/plain; charset=utf-8").String(http.StatusOK, "%s", id)
}

func (c *PasteController) Update(ctx frameworkhttp.Context) frameworkhttp.Response {
	id := ctx.Request().Route("id")
	body, err := readBody(ctx, facades.Config().GetInt("paste.max_bytes", 2*1024*1024))
	if err != nil {
		return ctx.Response().Status(http.StatusRequestEntityTooLarge).String("Paste too large")
	}

	if err := c.pastes.Update(id, body); err != nil {
		if err.Error() == "paste not found" {
			return ctx.Response().Status(http.StatusNotFound).String("Paste not found")
		}
		if strings.Contains(err.Error(), "maximum size") {
			return ctx.Response().Status(http.StatusRequestEntityTooLarge).String(err.Error())
		}
		return ctx.Response().Status(http.StatusBadRequest).String(err.Error())
	}

	return ctx.Response().Header("Content-Type", "text/plain; charset=utf-8").String(http.StatusOK, "%s", id)
}

func (c *PasteController) Destroy(ctx frameworkhttp.Context) frameworkhttp.Response {
	id := ctx.Request().Route("id")
	if !c.pastes.Delete(id) {
		return ctx.Response().Status(http.StatusNotFound).String("Paste not found")
	}
	return ctx.Response().Status(http.StatusOK).String("OK")
}

func readBody(ctx frameworkhttp.Context, maxBytes int) (string, error) {
	req := ctx.Request().Origin()
	var data []byte
	if req.Body != nil {
		limited := io.LimitReader(req.Body, int64(maxBytes)+1)
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

	// curl -d 'raw text' sends application/x-www-form-urlencoded; Goravel may
	// parse it as a form field name with an empty value.
	if req.Method == http.MethodPost || req.Method == http.MethodPut {
		if req.PostForm == nil {
			_ = req.ParseForm()
		}
		if body := bodyFromForm(req.PostForm); body != "" {
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
