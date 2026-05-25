package common

import "embed"

//go:embed cdn/codemirror/css/* cdn/codemirror/js/* cdn/iconify/* cdn/fonts/*
var CdnFS embed.FS
