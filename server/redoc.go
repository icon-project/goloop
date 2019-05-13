package server

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/labstack/echo/v4"
)

type RedocOpts struct {
	// BasePath for the UI path, defaults to: /
	BasePath string
	// Path combines with BasePath for the full UI path, defaults to: docs
	Path string
	// SpecURL the url to find the spec for
	SpecURL string
	// RedocURL for the js that generates the redoc site, defaults to: https://rebilly.github.io/ReDoc/releases/latest/redoc.min.js
	RedocURL string
	// Title for the documentation site, default to: API documentation
	Title string
}

func (r *RedocOpts) EnsureDefaults() {
	if r.BasePath == "" {
		r.BasePath = "/"
	}
	if r.Path == "" {
		r.Path = "docs"
	}
	if r.SpecURL == "" {
		r.SpecURL = "/swagger.yaml"
	}
	if r.RedocURL == "" {
		r.RedocURL = redocLatest
	}
	if r.Title == "" {
		r.Title = "API documentation"
	}
}

func Redoc(opts RedocOpts) echo.HandlerFunc {
	opts.EnsureDefaults()
	// pth := path.Join(opts.BasePath, opts.Path)
	tmpl := template.Must(template.New("redoc").Parse(redocTemplate))
	buf := bytes.NewBuffer(nil)
	_ = tmpl.Execute(buf, opts)
	b := buf.Bytes()
	return func(c echo.Context) error {
		return c.Blob(http.StatusOK, "text/html; charset=utf-8", b)
	}
}

const (
	redocLatest = "https://rebilly.github.io/ReDoc/releases/latest/redoc.min.js"
	// redocLatest = "https://cdn.jsdelivr.net/npm/redoc@next/bundles/redoc.standalone.js"

	redocTemplate = `<!DOCTYPE html>
<html>
  <head>
    <title>{{ .Title }}</title>
    <!-- needed for adaptive design -->
    <meta name="viewport" content="width=device-width, initial-scale=1">

    <!--
    ReDoc doesn't change outer page styles
    -->
    <style>
      body {
        margin: 0;
        padding: 0;
      }
    </style>
  </head>
  <body>
    <redoc spec-url='{{ .SpecURL }}'></redoc>
    <script src="{{ .RedocURL }}"> </script>
  </body>
</html>
`
)
