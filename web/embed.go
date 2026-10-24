// Package web embeds the semantic HTML5 templates and static CSS/JS assets
// into the Go binary using the standard library embed.FS.
package web

import "embed"

// TemplatesFS contains all HTML5 template files under web/templates/*.html.
//
//go:embed templates/*.html
var TemplatesFS embed.FS

// StaticFS contains all static CSS and JS files under web/static/**.
//
//go:embed static/css/* static/js/*
var StaticFS embed.FS
