package oauthapi

import (
	"embed"
	htmltemplate "html/template"
)

//go:embed templates/*.html.tmpl templates/*.js.tmpl
var templateFiles embed.FS

var (
	appOpenPageTemplate      = htmltemplate.Must(htmltemplate.ParseFS(templateFiles, "templates/app-open.html.tmpl", "templates/*.js.tmpl"))
	authorizeHandoffTemplate = htmltemplate.Must(htmltemplate.ParseFS(templateFiles, "templates/authorize-handoff.html.tmpl", "templates/*.js.tmpl"))
	deviceLoginPageTemplate  = htmltemplate.Must(htmltemplate.ParseFS(templateFiles, "templates/device-login.html.tmpl", "templates/*.js.tmpl"))
	deviceVerifyPageTemplate = htmltemplate.Must(htmltemplate.ParseFS(templateFiles, "templates/device-verify.html.tmpl", "templates/*.js.tmpl"))
)
