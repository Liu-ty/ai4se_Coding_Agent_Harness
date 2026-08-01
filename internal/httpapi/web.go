package httpapi

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed webdist/*
var webdist embed.FS

// WebHandler serves the Vite build embedded in the binary.
func WebHandler() http.Handler {
	assets, err := fs.Sub(webdist, "webdist")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(assets))
}
