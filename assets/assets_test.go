package assets_test

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/zackerydev/goth-template/assets"
)

func TestEmbeddedAssets(t *testing.T) {
	t.Parallel()
	filesystem := assets.FS()

	tests := []struct {
		path string
		want string
	}{
		{path: "css/app.css", want: "prefers-reduced-motion"},
		{path: "js/app.js", want: `htmx.config.noSwap`},
		{path: "js/htmx.min.js", want: "4.0.0"},
		{path: "js/htmx.LICENSE", want: "Zero-Clause BSD"},
	}

	for _, test := range tests {
		contents, err := fs.ReadFile(filesystem, test.path)
		if err != nil {
			t.Errorf("read embedded asset %q: %v", test.path, err)
			continue
		}
		if !strings.Contains(string(contents), test.want) {
			t.Errorf("embedded asset %q does not contain %q", test.path, test.want)
		}
	}
}
