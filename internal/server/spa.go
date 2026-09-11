package server

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/zackerydev/goth-template/assets"
)

func mountPublic(mux *http.ServeMux) {
	mux.Handle("GET /{$}", http.RedirectHandler("/contacts", http.StatusSeeOther))
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(assets.FS())))
	mux.Handle("GET /app", http.RedirectHandler("/app/", http.StatusSeeOther))
	mux.Handle("GET /app/", spaHandler())
}

type spaFileKind int

const (
	spaIndex spaFileKind = iota
	spaAsset
	spaMissing
)

func spaHandler() http.Handler {
	fsys := assets.SPA()
	files := http.StripPrefix("/app/", http.FileServerFS(fsys))
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		rel := strings.Trim(strings.TrimPrefix(request.URL.Path, "/app/"), "/")
		switch spaKind(fsys, rel) {
		case spaAsset:
			files.ServeHTTP(writer, request)
		case spaIndex:
			serveSPAIndex(writer, fsys)
		default:
			http.NotFound(writer, request)
		}
	})
}

func spaKind(fsys fs.FS, rel string) spaFileKind {
	if rel == "" || rel == "index.html" {
		return spaIndex
	}
	info, err := fs.Stat(fsys, rel)
	if err == nil && info.IsDir() {
		return spaMissing
	}
	if err == nil {
		return spaAsset
	}
	if strings.Contains(path.Base(rel), ".") {
		return spaMissing
	}
	return spaIndex
}

func serveSPAIndex(writer http.ResponseWriter, fsys fs.FS) {
	page, _ := fs.ReadFile(fsys, "index.html")
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = writer.Write(page)
}
