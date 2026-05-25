package mongodb

import (
	"embed"
)

//go:embed templates/*
var TemplatesFS embed.FS

//go:embed static/js/*.js static/css/*.css
var StaticFS embed.FS
