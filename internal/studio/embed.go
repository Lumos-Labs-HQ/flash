package studio

import "embed"

//go:embed templates/*.html
var TemplatesFS embed.FS

//go:embed static/js/*.js static/css/*.css static/image/*
var StaticFS embed.FS
