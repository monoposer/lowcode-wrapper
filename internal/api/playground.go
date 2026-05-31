package api

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed playground/static/*
var playgroundFS embed.FS

func RegisterPlayground(mux *http.ServeMux) {
	sub, _ := fs.Sub(playgroundFS, "playground/static")
	mux.Handle("/playground/", http.StripPrefix("/playground/", http.FileServer(http.FS(sub))))
	mux.HandleFunc("/playground", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/playground/", http.StatusMovedPermanently)
	})
}
