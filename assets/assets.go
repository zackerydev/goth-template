package assets

import (
	"embed"
	"io/fs"
)

// files contains the static files shipped with the application.
//
//go:embed css js
var files embed.FS

// FS returns the embedded static files.
func FS() fs.FS {
	return files
}
