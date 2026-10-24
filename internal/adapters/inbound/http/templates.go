// Package http implements the primary inbound HTTP adapter using Go's standard net/http,
// html/template with embedded files (web.TemplatesFS / web.StaticFS), Unpoly hypermedia
// conventions, and Server-Sent Events (SSE).
package http

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"

	"github.com/diego/go-refinery/web"
)

var (
	dashboardTmpl = mustParsePage("templates/dashboard.html")
	schemaTmpl    = mustParsePage("templates/schema.html")
	progressTmpl  = mustParsePage("templates/progress.html")
	recordsTmpl   = mustParsePage("templates/records.html")
)

func mustParsePage(pagePath string) *template.Template {
	funcs := template.FuncMap{
		"formatPct": func(pct float64) string {
			return fmt.Sprintf("%.1f%%", pct)
		},
	}
	t := template.New("layout.html").Funcs(funcs)
	template.Must(t.ParseFS(web.TemplatesFS, "templates/layout.html", pagePath))
	return t
}

func renderPage(w http.ResponseWriter, t *template.Template, status int, data any) {
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout.html", data); err != nil {
		http.Error(w, fmt.Sprintf("template rendering error: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}
