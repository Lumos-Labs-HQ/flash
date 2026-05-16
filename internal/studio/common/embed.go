package common

import "embed"

//go:embed cdn/codemirror/css/* cdn/codemirror/js/* cdn/ionicons/* cdn/iconify/* cdn/fonts/*
var CdnFS embed.FS
