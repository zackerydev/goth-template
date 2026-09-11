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

	shell, err := fs.ReadFile(assets.SPA(), "index.html")
	if err != nil {
		t.Fatalf("read SPA index: %v", err)
	}
	if !strings.Contains(string(shell), `id="root"`) {
		t.Fatalf("SPA index missing root: %s", shell)
	}
	if strings.Contains(string(shell), "/api/v1") {
		t.Fatal("SPA document leaked the JSON API path")
	}
}
