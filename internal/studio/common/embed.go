package common

import "embed"

//go:embed cdn/codemirror6/* cdn/iconify/* cdn/fonts/*
var CdnFS embed.FS
