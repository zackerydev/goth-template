package assets

import (
	"embed"
	"io/fs"
)

// files contains the static files shipped with the application.
//
//go:embed css js
var files embed.FS

//go:embed app
var spa embed.FS

// FS returns the embedded CSS and JavaScript files.
func FS() fs.FS {
	return files
}

// SPA returns the production React shell served at /app/.
func SPA() fs.FS {
	sub, _ := fs.Sub(spa, "app")
	return sub
}
