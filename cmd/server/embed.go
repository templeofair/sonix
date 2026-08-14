package main

import (
	"embed"
	"io/fs"
)

//go:embed static/*
var staticEmbed embed.FS

func staticFS() fs.FS {
	sub, _ := fs.Sub(staticEmbed, "static")
	return sub
}
